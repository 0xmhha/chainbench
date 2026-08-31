package testengine

import (
	"context"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// RunSpecFunc runs one parsed spec against a live environment, recording steps
// and assertions and returning the terminal status. It has the same shape as
// Deps.RunSpec so a wiring can be assigned to it directly.
type RunSpecFunc func(ctx context.Context, spec dsl.Spec, env session.Environment, rec session.TestRecord) (session.TestStatus, error)

// NewRunSpec is the production RunSpec wiring: it binds the DSL interpreter to
// the injected deps (typically with testhelper.Registry() so the built-in
// tx action and RPC assertions are available) and runs each spec against the
// environment. It is the composition boundary between the engine and the
// interpreter; keeping it thin is intentional — the behavior lives in the
// interpreter and the registered actions/assertions.
func NewRunSpec(deps dsl.Deps) RunSpecFunc {
	interp := dsl.NewInterpreter(deps)
	return func(ctx context.Context, spec dsl.Spec, env session.Environment, rec session.TestRecord) (session.TestStatus, error) {
		return interp.Run(ctx, spec, env, rec)
	}
}
