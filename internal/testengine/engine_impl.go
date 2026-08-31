package testengine

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// Deps injects the engine's collaborators so its orchestration is testable
// without real components. A production wiring composes place/keyreg/genesis/
// provision/launcher in BuildEnv and collector/interpreter in RunSpec.
type Deps struct {
	// NewSession creates the artifact session for one command. It takes a
	// context because a wiring may have to materialize key material (generating
	// identities shells out to an external binary) before the session is usable.
	NewSession func(ctx context.Context, command string) (session.Session, error)
	// Fingerprint derives an environment reuse key from a spec (resolved config
	// is applied by the wiring).
	Fingerprint func(spec testspec.Spec) session.Fingerprint
	// BuildEnv provisions and brings up a network for spec, returning the node
	// set and a teardown to run at the end of the session.
	BuildEnv func(ctx context.Context, env session.Environment, spec testspec.Spec) (node.NodeSet, TeardownFunc, error)
	// RunSpec starts collection and runs the interpreter for one test, recording
	// into rec.
	RunSpec func(ctx context.Context, spec testspec.Spec, env session.Environment, rec session.TestRecord) (session.TestStatus, error)
	// Applicable reports whether a spec applies to this run's target chain. Nil
	// means always applicable.
	Applicable func(spec testspec.Spec) bool
	// Command is the invoking command string recorded in session.json.
	Command string
	// Emit publishes an orchestration event (run/build/spec milestones) for the
	// dashboard. Nil disables emission — observation never affects the run.
	Emit func(ev collector.Event)
	// Network labels emitted events with the target chain/network. Optional.
	Network string
}

// engine is the concrete Engine.
type engine struct {
	deps Deps
}

// New returns an Engine over deps.
func New(deps Deps) Engine { return &engine{deps: deps} }

// emit publishes ev when a sink is wired, stamping the network label. It is a
// no-op when Deps.Emit is nil so observation never affects orchestration.
func (e *engine) emit(phase collector.Phase, kind collector.Kind, msg string, fields map[string]any) {
	if e.deps.Emit == nil {
		return
	}
	e.deps.Emit(collector.Event{
		Phase:   phase,
		Kind:    kind,
		Network: e.deps.Network,
		Message: msg,
		Fields:  fields,
	})
}

// Run executes the specs serially: parse, skip if inapplicable, reuse or build
// an environment by fingerprint, run the test, and record. Environments are torn
// down at the end. It returns the session root.
func (e *engine) Run(ctx context.Context, specs [][]byte) (string, error) {
	sess, err := e.deps.NewSession(ctx, e.deps.Command)
	if err != nil {
		return "", fmt.Errorf("engine: new session: %w", err)
	}

	var teardowns []TeardownFunc
	defer func() {
		for _, td := range teardowns {
			_ = td(ctx)
		}
	}()

	e.emit(collector.PhaseTest, collector.KindInfo, "run started", map[string]any{"specs": len(specs)})

	for i, raw := range specs {
		seq := i + 1
		spec, perr := testspec.Parse(raw)
		if perr != nil {
			rec := sess.Test(seq, fmt.Sprintf("spec-%d", seq))
			rec.Spec(raw)
			rec.Status(session.StatusBlocked)
			rec.Reason(perr.Error())
			e.emit(collector.PhaseTest, collector.KindError, "spec parse failed", map[string]any{"seq": seq, "error": perr.Error()})
			continue
		}

		rec := sess.Test(seq, spec.ID)
		rec.Spec(raw)

		if e.deps.Applicable != nil && !e.deps.Applicable(spec) {
			rec.Status(session.StatusSkip)
			// A skip with no reason reads as "this did not matter"; it usually
			// means a capability the target does not advertise.
			rec.Reason("does not apply to this target (chain or required capabilities)")
			e.emit(collector.PhaseTest, collector.KindInfo, "spec skipped", map[string]any{"seq": seq, "id": spec.ID})
			continue
		}

		env, ok := sess.Environment(e.deps.Fingerprint(spec))
		if !ok {
			var berr error
			env, berr = sess.NewEnvironment(e.deps.Fingerprint(spec))
			if berr != nil {
				rec.Status(session.StatusBlocked)
				rec.Reason(berr.Error())
				e.emit(collector.PhaseSetup, collector.KindError, "environment build failed", map[string]any{"seq": seq, "id": spec.ID, "error": berr.Error()})
				continue
			}
			e.emit(collector.PhaseSetup, collector.KindProgress, "building environment", map[string]any{"seq": seq, "id": spec.ID, "env": env.ID()})
			ns, td, buildErr := e.deps.BuildEnv(ctx, env, spec)
			if buildErr != nil {
				// The reason travels with the verdict: "blocked" on its own
				// sends the reader to the logs of a network that never started.
				rec.Status(session.StatusBlocked)
				rec.Reason(buildErr.Error())
				e.emit(collector.PhaseSetup, collector.KindError, "environment build failed", map[string]any{"seq": seq, "id": spec.ID, "env": env.ID(), "error": buildErr.Error()})
				continue
			}
			env.PopulateNodeTable(ns)
			_ = env.Save()
			if td != nil {
				teardowns = append(teardowns, td)
			}
		} else {
			e.emit(collector.PhaseSetup, collector.KindInfo, "environment reused", map[string]any{"seq": seq, "id": spec.ID, "env": env.ID()})
		}

		rec.SetEnvRef(env.ID())
		e.emit(collector.PhaseTest, collector.KindProgress, "running spec", map[string]any{"seq": seq, "id": spec.ID})
		st, runErr := e.deps.RunSpec(ctx, spec, env, rec)
		if runErr != nil {
			rec.Status(session.StatusFail)
			st = session.StatusFail
		}
		e.emit(collector.PhaseTest, collector.KindResult, "spec "+string(st), map[string]any{"seq": seq, "id": spec.ID, "status": string(st)})
	}

	e.emit(collector.PhaseTest, collector.KindResult, "run complete", nil)

	if err := sess.Save(); err != nil {
		return sess.Root(), fmt.Errorf("engine: save session: %w", err)
	}
	return sess.Root(), nil
}
