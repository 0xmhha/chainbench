package testspec

import (
	"context"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/keyreg"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// Deps are the collaborators an interpreter needs, injected at construction so
// no package-global state is used and tests stay isolated.
type Deps struct {
	Keys      keyreg.Registry
	Accounts  accounts.AccountProvider
	RPC       func(url string) *rpc.Client
	Collector collector.Collector
	Actions   Registry
}

// Registry holds action and assertion implementations, injected as an instance
// rather than a package global.
type Registry interface {
	Action(name string) (Action, bool)
	Assertion(name string) (Assertion, bool)
	RegisterAction(name string, a Action)
	RegisterAssertion(name string, a Assertion)
}

// Action is one atomic pre-action, step, or post-action (no partial success).
type Action interface {
	Do(ctx context.Context, ac *ActionCtx) error
}

// Assertion checks one expectation and returns its recorded result.
type Assertion interface {
	Check(ctx context.Context, ac *AssertCtx) (session.AssertResult, error)
}

// ActionCtx gives an action access to the environment, deps, and test record.
// Args are the DSL-boundary arguments, narrowed to typed values by Parse.
type ActionCtx struct {
	Env  session.Environment
	Deps *Deps
	Rec  session.TestRecord
	Args map[string]any
}

// AssertCtx gives an assertion access to the environment, deps, and targets.
type AssertCtx struct {
	Env  session.Environment
	Deps *Deps
	On   []node.Node
	Spec map[string]any
}

// Interpreter runs a parsed Spec against a running environment, recording each
// step and assertion, and returns the terminal status.
type Interpreter interface {
	Run(ctx context.Context, s Spec, env session.Environment, rec session.TestRecord) (session.TestStatus, error)
}

// registry is the default instance-scoped action/assertion registry.
type registry struct {
	actions    map[string]Action
	assertions map[string]Assertion
}

// NewRegistry returns an empty registry, optionally seeded with the built-in
// action and assertion sets.
func NewRegistry(withBuiltins bool) Registry {
	r := &registry{
		actions:    make(map[string]Action),
		assertions: make(map[string]Assertion),
	}
	// TODO(T3): seed built-in actions/assertions when withBuiltins is set.
	_ = withBuiltins
	return r
}

func (r *registry) Action(name string) (Action, bool) {
	a, ok := r.actions[name]
	return a, ok
}

func (r *registry) Assertion(name string) (Assertion, bool) {
	a, ok := r.assertions[name]
	return a, ok
}

func (r *registry) RegisterAction(name string, a Action) { r.actions[name] = a }

func (r *registry) RegisterAssertion(name string, a Assertion) { r.assertions[name] = a }

// interpreter is the default Deps-bound interpreter.
type interpreter struct {
	deps Deps
}

// NewInterpreter returns an interpreter bound to the given deps.
func NewInterpreter(d Deps) Interpreter {
	return &interpreter{deps: d}
}

func (i *interpreter) Run(ctx context.Context, s Spec, env session.Environment, rec session.TestRecord) (session.TestStatus, error) {
	return session.TestStatus(""), errNotImplemented
}
