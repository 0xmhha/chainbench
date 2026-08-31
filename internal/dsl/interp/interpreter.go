package interp

import (
	"context"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// Deps are the collaborators an interpreter needs, injected at construction so
// no package-global state is used and tests stay isolated.
type Deps struct {
	Keys      *store.KeySet
	Accounts  accounts.AccountProvider
	RPC       func(url string) *rpc.Client
	Collector collector.Collector
	Actions   Registry
	// Nodes controls node processes for fault-injection steps (stopNode /
	// startNode / restartNode). It is nil when the run does not own the node
	// processes — attach mode — and those actions then fail with a clear reason
	// rather than silently doing nothing.
	Nodes NodeControl
}

// Registry holds the action, assertion and reader implementations a run
// dispatches on, injected as an instance rather than a package global. The
// grammar and the interpreter know the names only as strings; what a name
// does is registered from outside (the testhelper module registers the
// built-ins), which is what keeps this package free of chain vocabulary.
type Registry interface {
	Action(name string) (Action, bool)
	Assertion(name string) (Assertion, bool)
	// Reader is a named source the "read" and "waitFor" actions draw from.
	// Registered beside the assertions so the two share one vocabulary — a
	// spec reads "chainId" with the same word it asserts it by.
	Reader(name string) (Reader, bool)
	RegisterAction(name string, a Action)
	RegisterAssertion(name string, a Assertion)
	RegisterReader(name string, r Reader)
}

// Reader reads one value from a target node for the spec's arguments.
type Reader func(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error)

// ActionRead is the one action name the grammar itself knows: a read names
// its source by string, and Unresolved checks that source against the
// registered readers offline.
const ActionRead = "read"

// NodeControl stops and restarts individual node processes. It is the boundary
// between the DSL and process management: the local engine wires an
// implementation backed by the launcher and procman, while attach mode leaves it
// nil because chainbench did not start those nodes and must not pretend it can
// stop them.
type NodeControl interface {
	// Stop terminates the node's process, verifying it is gone, and returns
	// the node as it now is (pid 0). The caller writes that back to the
	// environment's node table: the table is the one record of a pid.
	Stop(ctx context.Context, n node.Node) (node.Node, error)
	// Start relaunches a previously stopped node with its original arming and
	// returns it with its new pid.
	Start(ctx context.Context, n node.Node) (node.Node, error)
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
	// Hash and Receipt let a tx action surface its result so the interpreter can
	// record step provenance. They are outputs, set by the action.
	Hash    string
	Receipt map[string]any
	// Value is the action's result for step value binding: when the step declares
	// "save", the interpreter binds this under that name for later steps and
	// assertions to reference as "$name". An action that only sets Hash (a tx)
	// binds the hash, so submitting and then asserting on a transaction needs no
	// extra plumbing. It is an output, set by the action.
	Value any
	// Extra holds additional named bindings an action produces beyond the primary
	// "save" value — keyed by binding name. newAccount uses it to bind a freshly
	// generated private key under "saveKey" alongside the address under "save".
	// It is an output, set by the action; the interpreter merges it into the run
	// bindings after Value/Hash.
	Extra map[string]any
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
	Run(ctx context.Context, s dsl.Spec, env session.Environment, rec session.TestRecord) (session.TestStatus, error)
}

// registry is the default instance-scoped registry.
type registry struct {
	actions    map[string]Action
	assertions map[string]Assertion
	readers    map[string]Reader
}

// NewRegistry returns an empty registry. Nothing is seeded here: the built-in
// vocabulary lives in the testhelper module, and a caller registers it
// (testhelper.Register) or its own.
func NewRegistry() Registry {
	return &registry{
		actions:    make(map[string]Action),
		assertions: make(map[string]Assertion),
		readers:    make(map[string]Reader),
	}
}

func (r *registry) Action(name string) (Action, bool) {
	a, ok := r.actions[name]
	return a, ok
}

func (r *registry) Assertion(name string) (Assertion, bool) {
	a, ok := r.assertions[name]
	return a, ok
}

func (r *registry) Reader(name string) (Reader, bool) {
	rd, ok := r.readers[name]
	return rd, ok
}

func (r *registry) RegisterAction(name string, a Action) { r.actions[name] = a }

func (r *registry) RegisterAssertion(name string, a Assertion) { r.assertions[name] = a }

func (r *registry) RegisterReader(name string, rd Reader) { r.readers[name] = rd }

// interpreter is the default Deps-bound interpreter.
type interpreter struct {
	deps Deps
}

// NewInterpreter returns an interpreter bound to the given deps.
func NewInterpreter(d Deps) Interpreter {
	return &interpreter{deps: d}
}
