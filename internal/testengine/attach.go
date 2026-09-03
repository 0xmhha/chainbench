package testengine

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/testhelper"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// attachNetwork is the network label recorded for an attached NodeSet.
const attachNetwork = "attached"

// engineCommand is the invoking command recorded in session.json.
const engineCommand = "chainbench"

// busEmit returns an event sink publishing to bus, or nil when bus is nil so the
// engine's emission stays a no-op.
func busEmit(bus *collector.Bus) func(collector.Event) {
	if bus == nil {
		return nil
	}
	return bus.Publish
}

// AttachConfig configures an attach-mode Engine: it runs specs against an
// already-running network addressed by RPC URLs, building and launching nothing.
type AttachConfig struct {
	// Chain is the chain label (used for spec applicability and fingerprinting).
	Chain string
	// RPCURLs are the endpoints of the running nodes; the first is the primary.
	RPCURLs []string
	// ArtifactRoot is the base directory for session artifacts.
	ArtifactRoot string
	// Caps are extra capabilities the operator asserts the attached network
	// provides, beyond the implicit "rpc". An attached net that was launched
	// with a genesis overlay (e.g. account-extra, short-expiry) carries caps
	// that attach cannot detect from RPC alone, so the operator names them here
	// to let the gated specs run instead of skipping.
	Caps []string
	// Clock supplies the session start time; nil uses time.Now.
	Clock func() time.Time
	// Bus, when non-nil, receives orchestration events for the dashboard. Nil
	// disables emission.
	Bus *collector.Bus
	// NodeSet, when non-nil, is the full node table of the network being
	// attached to — a suite that composed the network itself passes the real
	// nodes (indices, hosts, every endpoint) instead of bare RPC URLs.
	NodeSet *node.NodeSet
	// Control, when non-nil, lets fault steps (stopNode/startNode/restartNode)
	// act on the node processes. Nil is plain attach's default: the run does
	// not own the processes, and those steps fail with a clear reason.
	Control interp.NodeControl
	// PreSpec, when non-nil, gates the network before each test (E6). A suite
	// that owns the network passes a gate that waits on or restarts unfit nodes;
	// plain attach leaves it nil (it does not own the processes to restart).
	PreSpec func(ctx context.Context, env session.Environment) error
	// OnFail, when non-nil, gathers failure evidence into a failed test's
	// observations/ (E8). A suite that owns the network passes a gatherer over
	// its nodes and workspace; plain attach leaves it nil.
	OnFail func(ctx context.Context, env session.Environment, rec session.TestRecord) error
	// LogReader, when non-nil, is how the collector reads node logs — a remote
	// target passes an SSH-backed reader so a remote node's log is captured (and
	// reconnected on a dropped session, E8); nil reads the local filesystem.
	LogReader collector.LogReader
}

// BuildEnvFunc provisions and brings up a network for a spec, returning the
// node set and a teardown. It has the same shape as Deps.BuildEnv so a wiring
// can be assigned to it directly.
type BuildEnvFunc func(ctx context.Context, env session.Environment, spec dsl.Spec) (node.NodeSet, TeardownFunc, error)

// NewAttachBuildEnv returns a BuildEnv that builds the node table from existing
// RPC endpoints without provisioning or launching anything. Its teardown is nil:
// attach did not create the nodes, so it must not stop them.
func NewAttachBuildEnv(chain string, eps []node.RPCEndpoint) BuildEnvFunc {
	return func(_ context.Context, _ session.Environment, _ dsl.Spec) (node.NodeSet, TeardownFunc, error) {
		ns, err := node.AttachedSet(chain, attachNetwork, eps)
		if err != nil {
			return node.NodeSet{}, nil, fmt.Errorf("engine: attach: %w", err)
		}
		return ns, nil, nil
	}
}

// NewAttachEngine composes an Engine that runs specs against a running network.
// No chain binary or preset is needed — only reachable RPC endpoints — so attach
// runs anywhere the endpoints are reachable.
func NewAttachEngine(cfg AttachConfig) (Engine, error) {
	if cfg.Chain == "" || cfg.ArtifactRoot == "" {
		return nil, fmt.Errorf("engine: attach config needs chain and artifactRoot")
	}
	if len(cfg.RPCURLs) == 0 {
		return nil, fmt.Errorf("engine: attach config needs at least one RPC URL")
	}
	eps := make([]node.RPCEndpoint, len(cfg.RPCURLs))
	for i, u := range cfg.RPCURLs {
		if u == "" {
			return nil, fmt.Errorf("engine: attach RPC URL %d is empty", i+1)
		}
		eps[i] = node.RPCEndpoint{RPCURL: u}
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	// The provider is what the interpreter signs and reads accounts with, and
	// resolving it here is also what rejects a chain the accounts SDK does not
	// know before a run gets far enough to fail obscurely. It no longer supplies
	// identity derivation, which keyring does in process.
	accts, err := accounts.ForChain(cfg.Chain)
	if err != nil {
		return nil, fmt.Errorf("engine: attach engine: %w", err)
	}

	run := NewRunSpec(interp.Deps{
		RPC:      func(u string) *rpc.Client { return rpc.Dial(u) },
		Actions:  testhelper.Registry(),
		Accounts: accts,
		Nodes:    cfg.Control,
	})

	build := NewAttachBuildEnv(cfg.Chain, eps)
	if cfg.NodeSet != nil {
		ns := *cfg.NodeSet
		build = func(context.Context, session.Environment, dsl.Spec) (node.NodeSet, TeardownFunc, error) {
			return ns, nil, nil
		}
	}

	return New(Deps{
		Command: engineCommand,
		NewSession: func(_ context.Context, cmd string) (session.Session, error) {
			// Attach owns no node identities, but a spec may still generate keys
			// mid-run, so the session gets a keyring rooted in its own keys/
			// directory rather than nothing.
			return session.New(cfg.ArtifactRoot, cmd, clock())
		},
		Fingerprint: func(s dsl.Spec) session.Fingerprint {
			return interp.Fingerprint(s, nodeconfig.Values{})
		},
		BuildEnv:   withCollection(build, cfg.Bus, nil, cfg.LogReader),
		RunSpec:    run,
		Applicable: applicableWithCaps(cfg.Chain, append([]string{attachCapability}, cfg.Caps...)),
		PreSpec:    cfg.PreSpec,
		OnFail:     cfg.OnFail,
		Emit:       busEmit(cfg.Bus),
		Network:    cfg.Chain,
	}), nil
}
