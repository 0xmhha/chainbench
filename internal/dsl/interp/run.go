package interp

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// Run executes a spec against a running environment: pre-actions (idempotent
// guards), steps (atomic), assertions, then post-actions (recorded but
// independent of the verdict). It records each step/assertion and the final
// status, and returns that status.
//
// Bindings are scoped to this call: a step that declares "save" binds its result
// for later steps and assertions to reference as "$name" (see binding.go). The
// scope is per-run, so nothing leaks between tests and the interpreter itself
// stays free of shared state.
func (i *interpreter) Run(ctx context.Context, s dsl.Spec, env session.Environment, rec session.TestRecord) (session.TestStatus, error) {
	if i.deps.Actions == nil {
		return session.StatusFail, fmt.Errorf("interp: interpreter has no action/assertion registry")
	}
	binds := Bindings{}

	// Pre-actions: a failure blocks the test (steps/assertions do not run).
	for _, pa := range s.PreActions {
		if err := i.runAction(ctx, pa, env, rec, binds); err != nil {
			rec.Status(session.StatusBlocked)
			return session.StatusBlocked, nil
		}
	}

	// The unified statement sequence: do statements fail fast (a broken step
	// invalidates everything after it), expect statements record and continue
	// (all verifications report). v1 specs desugar to [do... expect...], which
	// reproduces the historical steps-then-assertions behavior exactly; v2
	// interleaves freely (proposal G7).
	pass := true
	stepIdx := 0
	for _, st := range dsl.SequenceOf(s) {
		if st.Do != "" {
			if err := i.runStep(ctx, stepIdx, dsl.StatementStep(st), env, rec, binds); err != nil {
				// A failed do statement invalidates everything after it:
				// on-fail diagnostics run, post-actions do not (the v1
				// contract — cleanup assumes the steps it undoes happened).
				i.runRecorded(ctx, s.OnFailActions, env, rec, binds)
				rec.Status(session.StatusFail)
				return session.StatusFail, nil
			}
			stepIdx++
			continue
		}
		r, err := i.runAssertion(ctx, dsl.StatementAssertion(st), env, binds)
		rec.Assert(r)
		if err != nil || !r.Pass {
			pass = false
		}
	}
	status := session.StatusPass
	if !pass {
		status = session.StatusFail
		// On-fail hooks: diagnostics for a failed case, recorded like
		// post-actions.
		i.runRecorded(ctx, s.OnFailActions, env, rec, binds)
	}

	// Post-actions: recorded, but they do not change the verdict.
	i.runRecorded(ctx, s.PostActions, env, rec, binds)

	rec.Status(status)
	return status, nil
}

// runRecorded runs hook actions whose outcome is recorded but never changes
// the verdict.
func (i *interpreter) runRecorded(ctx context.Context, actions []map[string]any, env session.Environment, rec session.TestRecord, binds Bindings) {
	for _, a := range actions {
		name := dsl.ActionName(a)
		if err := i.runAction(ctx, a, env, rec, binds); err != nil {
			rec.PostAction(session.PostResult{Name: name, OK: false, Detail: err.Error()})
		} else {
			rec.PostAction(session.PostResult{Name: name, OK: true})
		}
	}
}

// runAction dispatches a single-key action entry ({name: args}) to the registry,
// substituting binding references in its args first.
func (i *interpreter) runAction(ctx context.Context, entry map[string]any, env session.Environment, rec session.TestRecord, binds Bindings) error {
	name := dsl.ActionName(entry)
	if name == "" {
		return fmt.Errorf("interp: empty action entry")
	}
	act, ok := i.deps.Actions.Action(name)
	if !ok {
		return fmt.Errorf("interp: unknown action %q", name)
	}
	args, err := resolveArgs(entry[name], binds)
	if err != nil {
		return err
	}
	ac := &ActionCtx{Env: env, Deps: &i.deps, Rec: rec, Args: args}
	if err := act.Do(ctx, ac); err != nil {
		return err
	}
	bindResult(binds, args, ac)
	return nil
}

// runStep runs a step action and records a StepResult (including any tx
// hash/receipt the action surfaces) even on failure.
func (i *interpreter) runStep(ctx context.Context, idx int, entry map[string]any, env session.Environment, rec session.TestRecord, binds Bindings) error {
	name := dsl.ActionName(entry)
	act, ok := i.deps.Actions.Action(name)
	if !ok {
		on, _ := dsl.ArgsOf(entry[name])["on"].(string)
		err := fmt.Errorf("interp: unknown action %q", name)
		if name == "" {
			err = fmt.Errorf("interp: empty action entry")
		}
		rec.Step(idx, session.StepResult{Index: idx, Type: name, On: on, Error: err.Error()})
		return err
	}
	args, err := resolveArgs(entry[name], binds)
	if err != nil {
		// An unbound reference is a step failure, recorded like any other: the
		// step never runs, so there is no hash or receipt to report.
		on, _ := dsl.ArgsOf(entry[name])["on"].(string)
		rec.Step(idx, session.StepResult{Index: idx, Type: name, On: on, Error: err.Error()})
		return err
	}
	on, _ := args["on"].(string)
	ac := &ActionCtx{Env: env, Deps: &i.deps, Rec: rec, Args: args}
	runErr := act.Do(ctx, ac)
	step := session.StepResult{Index: idx, Type: name, On: on, Hash: ac.Hash, Receipt: ac.Receipt}
	if runErr != nil {
		step.Error = runErr.Error()
	}
	rec.Step(idx, step)
	if runErr != nil {
		return runErr
	}
	bindResult(binds, args, ac)
	return nil
}

// runAssertion dispatches an assertion entry (its "assert" field names the
// registered assertion) and returns its recorded result. Binding references in
// the entry are substituted first, so an assertion can compare against a value
// an earlier step saved.
func (i *interpreter) runAssertion(ctx context.Context, entry map[string]any, env session.Environment, binds Bindings) (session.AssertResult, error) {
	name, _ := entry["assert"].(string)
	if name == "" {
		return session.AssertResult{Pass: false, Provenance: entry}, fmt.Errorf(`interp: assertion missing "assert"`)
	}
	as, ok := i.deps.Actions.Assertion(name)
	if !ok {
		return session.AssertResult{Assert: name, Pass: false}, fmt.Errorf("interp: unknown assertion %q", name)
	}
	resolved, err := resolveRefs(entry, binds)
	if err != nil {
		return session.AssertResult{Assert: name, Pass: false, Provenance: entry, Actual: err.Error()}, err
	}
	spec := dsl.ArgsOf(resolved)
	r, err := as.Check(ctx, &AssertCtx{Env: env, Deps: &i.deps, On: i.resolveOn(spec, env), Spec: spec})
	r.Assert = name
	return r, err
}

// resolveArgs substitutes binding references in an action's argument value and
// returns it as an args map.
func resolveArgs(v any, binds Bindings) (map[string]any, error) {
	resolved, err := resolveRefs(v, binds)
	if err != nil {
		return nil, err
	}
	return dsl.ArgsOf(resolved), nil
}

// bindResult stores an action's result under the name its args declare via
// "save". An action that sets no explicit Value binds its tx hash, so a plain
// sendTx step is referenceable without the action knowing about bindings.
func bindResult(binds Bindings, args map[string]any, ac *ActionCtx) {
	for k, v := range ac.Extra {
		binds[k] = v
	}
	name := saveName(args)
	if name == "" {
		return
	}
	if ac.Value != nil {
		binds[name] = ac.Value
		return
	}
	if ac.Hash != "" {
		binds[name] = ac.Hash
	}
}

// resolveOn resolves the entry's "on" (single) or "onEach" ([]) selectors to
// nodes, best-effort (an unresolved selector yields no nodes here; the
// assertion decides how to treat that).
func (i *interpreter) resolveOn(entry map[string]any, env session.Environment) []node.Node {
	if sel, ok := entry["on"].(string); ok {
		if n, err := env.Resolve(sel); err == nil {
			return []node.Node{n}
		}
		return nil
	}
	if raw, ok := entry["onEach"].([]any); ok {
		sels := make([]string, 0, len(raw))
		for _, s := range raw {
			if str, ok := s.(string); ok {
				sels = append(sels, str)
			}
		}
		if nodes, err := env.ResolveEach(sels); err == nil {
			return nodes
		}
	}
	return nil
}
