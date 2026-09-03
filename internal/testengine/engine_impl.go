package testengine

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/report"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
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
	Fingerprint func(spec dsl.Spec) session.Fingerprint
	// BuildEnv provisions and brings up a network for spec, returning the node
	// set and a teardown to run at the end of the session.
	BuildEnv func(ctx context.Context, env session.Environment, spec dsl.Spec) (node.NodeSet, TeardownFunc, error)
	// RunSpec starts collection and runs the interpreter for one test, recording
	// into rec.
	RunSpec func(ctx context.Context, spec dsl.Spec, env session.Environment, rec session.TestRecord) (session.TestStatus, error)
	// Applicable reports whether a spec applies to this run's target chain. Nil
	// means always applicable.
	Applicable func(spec dsl.Spec) bool
	// PreSpec gates the environment right before each test runs on it (E6): a
	// network left unfit by a prior fault test is waited on or restarted within
	// limits, and a state needing a destructive remedy blocks the test. A
	// non-nil error blocks the test with that reason. Nil means no gate.
	PreSpec func(ctx context.Context, env session.Environment) error
	// OnFail gathers failure evidence (node logs, process, RPC/block snapshot)
	// into the test's observations/ when a test ends FAIL or BLOCKED (E8). It is
	// best-effort: an error is emitted, never fatal, since it runs after the
	// verdict is already decided. Nil means no gathering.
	OnFail func(ctx context.Context, env session.Environment, rec session.TestRecord) error
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
		spec, perr := dsl.Parse(raw)
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
		if e.deps.PreSpec != nil {
			if gerr := e.deps.PreSpec(ctx, env); gerr != nil {
				rec.Status(session.StatusBlocked)
				rec.Reason(gerr.Error())
				e.emit(collector.PhaseSetup, collector.KindError, "pre-test gate failed", map[string]any{"seq": seq, "id": spec.ID, "env": env.ID(), "error": gerr.Error()})
				continue
			}
		}
		e.emit(collector.PhaseTest, collector.KindProgress, "running spec", map[string]any{"seq": seq, "id": spec.ID})
		st, runErr := e.deps.RunSpec(ctx, spec, env, rec)
		if runErr != nil {
			rec.Status(session.StatusFail)
			st = session.StatusFail
		}
		// A failed or blocked test gets its evidence gathered before the run moves
		// on: the network it failed against may be gone or changed by the next
		// test. Best-effort — the verdict already stands.
		if st == session.StatusFail || st == session.StatusBlocked {
			if e.deps.OnFail != nil {
				if ferr := e.deps.OnFail(ctx, env, rec); ferr != nil {
					e.emit(collector.PhaseTest, collector.KindError, "failure-data collection", map[string]any{"seq": seq, "id": spec.ID, "error": ferr.Error()})
				}
			}
		}
		e.emit(collector.PhaseTest, collector.KindResult, "spec "+string(st), map[string]any{"seq": seq, "id": spec.ID, "status": string(st)})
	}

	e.emit(collector.PhaseTest, collector.KindResult, "run complete", nil)

	if err := sess.Save(); err != nil {
		return sess.Root(), fmt.Errorf("engine: save session: %w", err)
	}
	// The verdicts are now persisted; the report builder aggregates them into
	// report.json for the CLI/MCP report surfaces. A report failure does not
	// invalidate the run, so it is surfaced but not fatal to Run's result.
	if _, err := report.Generate(sess.Root()); err != nil {
		return sess.Root(), fmt.Errorf("engine: build report: %w", err)
	}
	return sess.Root(), nil
}
