package engine

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/keyreg"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/procman"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/core/supervisor"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// busEmit returns an event sink publishing to bus, or nil when bus is nil so the
// engine's emission stays a no-op.
func busEmit(bus *obs.Bus) func(obs.Event) {
	if bus == nil {
		return nil
	}
	return bus.Publish
}

// Local engine defaults.
const (
	defaultValidators    = 4
	defaultHealthTimeout = 90 * time.Second
	defaultPortBand      = 100

	localP2PBase  = 31000
	localRPCBase  = 8600
	localPortStep = 10

	engineCommand = "chainbench"
)

// LocalConfig configures a single-chain, local-host Engine: the whole pipeline
// composed for one chain running on 127.0.0.1 from a preset key set.
type LocalConfig struct {
	// Chain is the registry chain id (e.g. "stablenet").
	Chain string
	// Binary is the node executable path.
	Binary string
	// KeysDir is the preset directory (metadata.json + node<i>/ + password).
	// Ignored when Keys is set.
	KeysDir string
	// Keys selects where the node identities come from — an existing set, or a
	// freshly generated one (algorithm steps 2-3). Nil uses
	// PresetKeySource{KeysDir}, the reproducible default.
	Keys KeySource
	// ArtifactRoot is the base directory for session artifacts.
	ArtifactRoot string
	// Validators is the validator node count; <=0 uses the default.
	Validators int
	// Clock supplies the session start time; nil uses time.Now (injected so
	// tests are deterministic).
	Clock func() time.Time
	// Bus, when non-nil, receives orchestration events for the dashboard. Nil
	// disables emission.
	Bus *obs.Bus
}

// NewLocalEngine composes a runnable Engine for one local chain: it wires the
// allocator, preset genesis source, local launcher, block-advance health gate,
// interpreter, and session store into engine.Deps. It is the top-level assembly
// the CLI/MCP entrypoints call — the seam where the live-proven components come
// together behind Engine.Run.
func NewLocalEngine(cfg LocalConfig) (Engine, error) {
	if cfg.Chain == "" || cfg.Binary == "" || cfg.ArtifactRoot == "" {
		return nil, fmt.Errorf("engine: local config needs chain, binary, and artifactRoot")
	}
	keySrc := cfg.Keys
	if keySrc == nil {
		if cfg.KeysDir == "" {
			return nil, fmt.Errorf("engine: local config needs keysDir or a key source")
		}
		keySrc = PresetKeySource{Path: cfg.KeysDir}
	}
	keysDir := keySrc.Dir()
	plugin, err := registry.Get(cfg.Chain)
	if err != nil {
		return nil, fmt.Errorf("engine: local engine: %w", err)
	}
	validators := cfg.Validators
	if validators <= 0 {
		validators = defaultValidators
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	// The registry derives an address for every key it takes in, which is how
	// RegisterIdentities checks a key set's declared identities against its key
	// material. Chain-specific because address derivation is a protocol fact.
	accts, err := accounts.ForChain(cfg.Chain)
	if err != nil {
		return nil, fmt.Errorf("engine: local engine: %w", err)
	}
	keyDeps := keyreg.Deps{DeriveAddress: accts.AddressForKey}

	// The controller fronts the launcher so a fault step can reach an individual
	// node process later; it shares the supervisor's procman so a node stopped
	// and restarted mid-test is still torn down at the end.
	procs := procman.New()
	controller := NewNodeController(LocalLauncher{Plugin: plugin, Binary: cfg.Binary, KeysDir: keysDir}, procs)
	sup := supervisor.New(supervisor.Deps{
		Launch:     controller.Launch,
		HealthGate: NewBlockAdvanceGate(1, defaultHealthTimeout),
		Procman:    procs,
	})
	build := NewBuildEnv(BuildDeps{
		Plugin:     plugin,
		Allocator:  place.New(place.Config{P2PBase: localP2PBase, P2PStep: localPortStep, RPCBase: localRPCBase, RPCStep: localPortStep}),
		Genesis:    PresetGenesisSource{KeysDir: keysDir},
		Supervisor: sup,
		Mode:       place.LocalStepped,
		Capacity:   place.Capacity{MinValidators: 1, PortBandSize: defaultPortBand},
		Caps:       []string{"ws"},
		Reqs:       validatorReqs(validators),
	})
	run := NewRunSpec(testspec.Deps{
		RPC:     func(u string) *rpc.Client { return rpc.Dial(u) },
		Actions: testspec.NewRegistry(true),
		Nodes:   controller,
	})

	return New(Deps{
		Command: engineCommand,
		NewSession: func(ctx context.Context, cmd string) (session.Session, error) {
			// Materialize the identities first: generating them can fail (no
			// bootnode binary) and there is no point creating a session tree for
			// a run that cannot start.
			ks, err := keySrc.Ensure(ctx, validators)
			if err != nil {
				return nil, err
			}
			sess, err := session.NewWithKeys(cfg.ArtifactRoot, cmd, clock(), keyDeps)
			if err != nil {
				return nil, err
			}
			if err := RegisterIdentities(ctx, sess.Keys(), ks, validators); err != nil {
				return nil, err
			}
			return sess, nil
		},
		Fingerprint: func(s testspec.Spec) session.Fingerprint {
			return s.Fingerprint(config.Values{})
		},
		BuildEnv:   withCollection(build, cfg.Bus, nil),
		RunSpec:    run,
		Applicable: applicableWithCaps(cfg.Chain, localCapabilities(plugin)),
		Emit:       busEmit(cfg.Bus),
		Network:    cfg.Chain,
	}), nil
}

// validatorReqs returns a Reqs function producing n validator placement
// requests. The per-node binary is left empty so AssemblePlan falls back to the
// manifest binary; the launcher overrides it with the resolved path.
func validatorReqs(n int) func(testspec.Spec) []place.NodeReq {
	return func(testspec.Spec) []place.NodeReq {
		reqs := make([]place.NodeReq, n)
		for i := range reqs {
			reqs[i] = place.NodeReq{Name: fmt.Sprintf("node%d", i+1), Role: node.RoleValidator}
		}
		return reqs
	}
}

// applicableTo reports whether a spec applies to chain: an empty or absent
// applicableChains applies to every chain; otherwise chain must appear in the
// comma/space-separated list.
func applicableTo(chain string) func(testspec.Spec) bool {
	return func(s testspec.Spec) bool {
		list := strings.FieldsFunc(s.ApplicableChains, func(r rune) bool { return r == ',' || r == ' ' })
		if len(list) == 0 {
			return true
		}
		for _, c := range list {
			if c == chain {
				return true
			}
		}
		return false
	}
}
