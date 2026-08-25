package testengine

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// attachNetwork is the network label recorded for an attached NodeSet.
const attachNetwork = "attached"

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
	Bus *obs.Bus
}

// NewAttachBuildEnv returns a BuildEnv that builds the node table from existing
// RPC endpoints without provisioning or launching anything. Its teardown is nil:
// attach did not create the nodes, so it must not stop them.
func NewAttachBuildEnv(chain string, eps []node.RPCEndpoint) BuildEnvFunc {
	return func(_ context.Context, _ session.Environment, _ testspec.Spec) (node.NodeSet, TeardownFunc, error) {
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

	run := NewRunSpec(testspec.Deps{
		RPC:      func(u string) *rpc.Client { return rpc.Dial(u) },
		Actions:  testspec.NewRegistry(true),
		Accounts: accts,
	})

	return New(Deps{
		Command: engineCommand,
		NewSession: func(_ context.Context, cmd string) (session.Session, error) {
			// Attach owns no node identities, but a spec may still generate keys
			// mid-run, so the session gets a keyring rooted in its own keys/
			// directory rather than nothing.
			return session.New(cfg.ArtifactRoot, cmd, clock())
		},
		Fingerprint: func(s testspec.Spec) session.Fingerprint {
			return s.Fingerprint(config.Values{})
		},
		BuildEnv:   withCollection(NewAttachBuildEnv(cfg.Chain, eps), cfg.Bus, nil),
		RunSpec:    run,
		Applicable: applicableWithCaps(cfg.Chain, append([]string{attachCapability}, cfg.Caps...)),
		Emit:       busEmit(cfg.Bus),
		Network:    cfg.Chain,
	}), nil
}
