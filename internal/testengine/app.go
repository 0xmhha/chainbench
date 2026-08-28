package testengine

import (
	"context"
	"fmt"
	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/testhelper"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/launcher"
	"github.com/0xmhha/chainbench/internal/core/launchopt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
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
	// minValidators is the BFT floor the allocator checks before placing.
	minValidators = 1

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
	// ChainID, when non-zero, overrides the manifest chain id in the built
	// genesis. The network id follows the chain id unless NetworkID also set.
	ChainID int64
	// NetworkID, when non-zero, pins the devp2p network id on every node's
	// command line (--networkid).
	NetworkID int64
	// LaunchOverrides are high-precedence launch knobs applied to every node's
	// argv through the launchopt Builder (env.launch / case layers).
	LaunchOverrides []launchopt.Override
	// Pool decides the port bands and the capacity bound. Its zero value is
	// the built-in local plan; a caller that read the operator's server set
	// passes that server's pool, which is how site-specific ports reach a run
	// without ever entering a spec.
	Pool resource.Pool
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
// the CLI/MCP entrypoints call — the boundary where the live-proven components come
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
	// The provider is what the interpreter signs and reads accounts with, and
	// resolving it here is also what rejects a chain the accounts SDK does not
	// know before a run gets far enough to fail obscurely. It no longer supplies
	// identity derivation, which keyring does in process.
	accts, err := accounts.ForChain(cfg.Chain)
	if err != nil {
		return nil, fmt.Errorf("engine: local engine: %w", err)
	}

	// The controller fronts the launcher so a fault step can reach an individual
	// node process later; it shares the launcher's procman so a node stopped
	// and restarted mid-test is still torn down at the end.
	// A pinned network id rides the same override layer as user launch knobs,
	// so every node's argv carries it uniformly.
	overrides := cfg.LaunchOverrides
	if cfg.NetworkID != 0 {
		overrides = append([]launchopt.Override{
			{Key: launchopt.KeyNetworkID, Value: strconv.FormatInt(cfg.NetworkID, 10), Layer: launchopt.LayerEnv},
		}, overrides...)
	}
	procs := process.New()
	controller := NewNodeController(LocalLauncher{
		Plugin: plugin, Binary: cfg.Binary, KeysDir: keysDir, LaunchOverrides: overrides,
	}, procs)
	sup := launcher.New(launcher.Deps{
		Launch:     controller.Launch,
		HealthGate: NewBlockAdvanceGate(1, defaultHealthTimeout),
		Action:     WemixBootstrap{Binary: cfg.Binary, KeysDir: keysDir}.Action,
		Procman:    procs,
	})
	pool := cfg.Pool
	if pool.Source == "" {
		pool = resource.Builtin(minValidators, defaultPortBand)
	}
	pool.Reservation = plugin.Family().PortReservation()
	// A family whose genesis is generated by its binary needs a different
	// source and a bootstrap between phases; one whose genesis is a template
	// with the validator set written in needs neither.
	genesisSource := genesis.SourceFor(plugin, genesis.Config{KeysDir: keysDir, Binary: cfg.Binary, ChainID: cfg.ChainID})
	var provisionExtra func(context.Context, driver.Plan, map[string][]byte) error
	if plugin.Family().ID() == poa.FamilyID {
		provisionExtra = writeExtra
	}

	build := NewBuildEnv(BuildDeps{
		Plugin:         plugin,
		Pool:           pool,
		Genesis:        genesisSource,
		Supervisor:     sup,
		Caps:           []string{"ws"},
		Reqs:           validatorReqs(validators),
		ProvisionExtra: provisionExtra,
	})
	run := NewRunSpec(testspec.Deps{
		RPC:      func(u string) *rpc.Client { return rpc.Dial(u) },
		Actions:  testhelper.Registry(),
		Nodes:    controller,
		Accounts: accts,
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
			sess, err := session.New(cfg.ArtifactRoot, cmd, clock())
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
func validatorReqs(n int) func(testspec.Spec) []node.LaunchReq {
	return func(testspec.Spec) []node.LaunchReq {
		reqs := make([]node.LaunchReq, n)
		for i := range reqs {
			reqs[i] = node.LaunchReq{Role: node.RoleValidator}
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

// writeExtra puts the genesis step's by-products on the target beside the
// genesis, under the names a later step reads them by.
func writeExtra(ctx context.Context, plan driver.Plan, files map[string][]byte) error {
	store := filestore.Local{}
	for name, content := range files {
		if err := store.Write(ctx, filepath.Join(plan.DataRoot, name), content, 0o644); err != nil {
			return fmt.Errorf("engine: write %s: %w", name, err)
		}
	}
	return nil
}
