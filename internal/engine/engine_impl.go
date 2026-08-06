package engine

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// TeardownFunc stops and cleans up an environment built during a run.
type TeardownFunc func(ctx context.Context) error

// Deps injects the engine's collaborators so its orchestration is testable
// without real components. A production wiring composes place/keyreg/genesis/
// provision/supervisor in BuildEnv and collector/interpreter in RunSpec.
type Deps struct {
	// NewSession creates the artifact session for one command.
	NewSession func(command string) (session.Session, error)
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
}

// engine is the concrete Engine.
type engine struct {
	deps Deps
}

// New returns an Engine over deps.
func New(deps Deps) Engine { return &engine{deps: deps} }

// Run executes the specs serially: parse, skip if inapplicable, reuse or build
// an environment by fingerprint, run the test, and record. Environments are torn
// down at the end. It returns the session root.
func (e *engine) Run(ctx context.Context, specs [][]byte) (string, error) {
	sess, err := e.deps.NewSession(e.deps.Command)
	if err != nil {
		return "", fmt.Errorf("engine: new session: %w", err)
	}

	var teardowns []TeardownFunc
	defer func() {
		for _, td := range teardowns {
			_ = td(ctx)
		}
	}()

	for i, raw := range specs {
		seq := i + 1
		spec, perr := testspec.Parse(raw)
		if perr != nil {
			rec := sess.Test(seq, fmt.Sprintf("spec-%d", seq))
			rec.Spec(raw)
			rec.Status(session.StatusBlocked)
			continue
		}

		rec := sess.Test(seq, spec.ID)
		rec.Spec(raw)

		if e.deps.Applicable != nil && !e.deps.Applicable(spec) {
			rec.Status(session.StatusSkip)
			continue
		}

		env, ok := sess.Environment(e.deps.Fingerprint(spec))
		if !ok {
			var berr error
			env, berr = sess.NewEnvironment(e.deps.Fingerprint(spec))
			if berr != nil {
				rec.Status(session.StatusBlocked)
				continue
			}
			ns, td, buildErr := e.deps.BuildEnv(ctx, env, spec)
			if buildErr != nil {
				rec.Status(session.StatusBlocked)
				continue
			}
			env.PopulateNodeTable(ns)
			_ = env.Save()
			if td != nil {
				teardowns = append(teardowns, td)
			}
		}

		rec.SetEnvRef(env.ID())
		if _, runErr := e.deps.RunSpec(ctx, spec, env, rec); runErr != nil {
			rec.Status(session.StatusFail)
		}
	}

	if err := sess.Save(); err != nil {
		return sess.Root(), fmt.Errorf("engine: save session: %w", err)
	}
	return sess.Root(), nil
}
