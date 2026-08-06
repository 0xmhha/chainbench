package testspec

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// Run executes a spec against a running environment: pre-actions (idempotent
// guards), steps (atomic), assertions, then post-actions (recorded but
// independent of the verdict). It records each step/assertion and the final
// status, and returns that status.
func (i *interpreter) Run(ctx context.Context, s Spec, env session.Environment, rec session.TestRecord) (session.TestStatus, error) {
	if i.deps.Actions == nil {
		return session.StatusFail, fmt.Errorf("testspec: interpreter has no action/assertion registry")
	}

	// Pre-actions: a failure blocks the test (steps/assertions do not run).
	for _, pa := range s.PreActions {
		if err := i.runAction(ctx, pa, env, rec); err != nil {
			rec.Status(session.StatusBlocked)
			return session.StatusBlocked, nil
		}
	}

	// Steps: a failed step expectation fails the test.
	for idx, st := range s.Steps {
		if err := i.runStep(ctx, idx, st, env, rec); err != nil {
			rec.Status(session.StatusFail)
			return session.StatusFail, nil
		}
	}

	// Assertions: all must pass.
	pass := true
	for _, as := range s.Assertions {
		r, err := i.runAssertion(ctx, as, env)
		rec.Assert(r)
		if err != nil || !r.Pass {
			pass = false
		}
	}
	status := session.StatusPass
	if !pass {
		status = session.StatusFail
	}

	// Post-actions: recorded, but they do not change the verdict.
	for _, poa := range s.PostActions {
		name := actionName(poa)
		if err := i.runAction(ctx, poa, env, rec); err != nil {
			rec.PostAction(session.PostResult{Name: name, OK: false, Detail: err.Error()})
		} else {
			rec.PostAction(session.PostResult{Name: name, OK: true})
		}
	}

	rec.Status(status)
	return status, nil
}

// runAction dispatches a single-key action entry ({name: args}) to the registry.
func (i *interpreter) runAction(ctx context.Context, entry map[string]any, env session.Environment, rec session.TestRecord) error {
	name := actionName(entry)
	if name == "" {
		return fmt.Errorf("testspec: empty action entry")
	}
	act, ok := i.deps.Actions.Action(name)
	if !ok {
		return fmt.Errorf("testspec: unknown action %q", name)
	}
	return act.Do(ctx, &ActionCtx{Env: env, Deps: &i.deps, Rec: rec, Args: argsOf(entry[name])})
}

// runStep runs a step action and records a StepResult, even on failure.
func (i *interpreter) runStep(ctx context.Context, idx int, entry map[string]any, env session.Environment, rec session.TestRecord) error {
	name := actionName(entry)
	err := i.runAction(ctx, entry, env, rec)
	on, _ := argsOf(entry[name])["on"].(string)
	rec.Step(idx, session.StepResult{Index: idx, Type: name, On: on})
	return err
}

// runAssertion dispatches an assertion entry (its "assert" field names the
// registered assertion) and returns its recorded result.
func (i *interpreter) runAssertion(ctx context.Context, entry map[string]any, env session.Environment) (session.AssertResult, error) {
	name, _ := entry["assert"].(string)
	if name == "" {
		return session.AssertResult{Pass: false, Provenance: entry}, fmt.Errorf(`testspec: assertion missing "assert"`)
	}
	as, ok := i.deps.Actions.Assertion(name)
	if !ok {
		return session.AssertResult{Assert: name, Pass: false}, fmt.Errorf("testspec: unknown assertion %q", name)
	}
	r, err := as.Check(ctx, &AssertCtx{Env: env, Deps: &i.deps, On: i.resolveOn(entry, env), Spec: entry})
	r.Assert = name
	return r, err
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

// actionName returns the single key of an action entry. Action entries are
// {name: args} by schema; an empty entry yields "".
func actionName(entry map[string]any) string {
	for k := range entry {
		return k
	}
	return ""
}

// argsOf returns the arguments map for an action value, or an empty map when the
// value is not a map (e.g. a bare boolean like {"ensureChain": true}).
func argsOf(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}
