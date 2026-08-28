package chainsetup

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// LocalSetup provisions and launches a local-host network from a resolved plan,
// sourcing the network genesis from a preset. It is the CLI/MCP entrypoint that
// composes BuildLocalPlan (the fixed-base-port plan) with PresetGenesisSource
// (genesis, overrides/overlay applied) and LocalLauncher (arm → materialize →
// init → launch), replacing the retired pipeline/setup Launch/Provision surface.
//
// A remote host is driven by setting Driver to a RemoteDriver and Sink to a
// driver.RemoteFileStore over the same transport: the launcher then ships the
// genesis, per-node config, and preset identities to that host.
type LocalSetup struct {
	// Plugin is the target chain (genesis template, RPC namespace, binary name).
	Plugin registry.ChainPlugin
	// Config is the resolved config; its genesis.overrides.* and genesis.overlay
	// keys shape the built genesis.
	Config config.Values
	// KeysDir is the preset key set (metadata.json + node<i>/ identities +
	// password). Callers should pass an absolute path so the launch argv's
	// identity paths resolve independently of the node process cwd.
	KeysDir string
	// Binary is the resolved node executable path (required by Launch; unused by
	// Provision).
	Binary string
	// Driver launches the nodes; nil defaults to the local driver.
	Driver driver.Driver
	// Files materializes on-disk files; nil defaults to the local filesystem. Set a
	// driver.RemoteFileStore to ship genesis + config to a remote host.
	Files filestore.Store
	// Bus receives setup progress; nil drops the events. It exists because the
	// dashboard shows a bring-up as it happens, and a launch that reports
	// nothing until it finishes looks indistinguishable from one that hung.
	Bus *obs.Bus
}

// Launch attaches the network genesis to plan and brings every node up through
// LocalLauncher, returning the running node set and the armed specs (so the
// caller can persist them and later relaunch a single node). It is the engine
// equivalent of the retired setup.LaunchWithSpecs.
func (s LocalSetup) Launch(ctx context.Context, plan driver.Plan) (node.NodeSet, []driver.NodeSpec, error) {
	if s.Binary == "" {
		return node.NodeSet{}, nil, fmt.Errorf("engine: local launch needs a resolved binary path")
	}
	if err := s.attachGenesis(ctx, &plan); err != nil {
		return node.NodeSet{}, nil, err
	}
	s.emit(obs.Event{Phase: obs.PhaseSetup, Kind: obs.KindProgress, Network: plan.Network,
		Message: "launching", Fields: map[string]any{"nodes": len(plan.Nodes)}})
	res, specs, err := s.launcher().LaunchArmed(ctx, plan, nil)
	if err != nil {
		s.emit(obs.Event{Phase: obs.PhaseSetup, Kind: obs.KindError, Network: plan.Network,
			Message: "launch failed", Fields: map[string]any{"error": err.Error()}})
		return res.Nodes, specs, err
	}
	s.emit(obs.Event{Phase: obs.PhaseSetup, Kind: obs.KindProgress, Network: plan.Network,
		Message: "nodes launched", Fields: map[string]any{"nodes": len(res.Nodes.Nodes)}})
	return res.Nodes, specs, err
}

// emit publishes an event when a bus is attached.
func (s LocalSetup) emit(e obs.Event) {
	if s.Bus != nil {
		s.Bus.Publish(e)
	}
}

// Provision attaches the network genesis to plan and materializes its on-disk
// artifacts (genesis, per-node config, and — for a remote driver — the shipped
// identities) without initializing datadirs or launching. It is the engine
// equivalent of the retired setup.Provision.
func (s LocalSetup) Provision(ctx context.Context, plan driver.Plan) ([]driver.NodeSpec, error) {
	if err := s.attachGenesis(ctx, &plan); err != nil {
		return nil, err
	}
	specs, err := s.launcher().Provision(ctx, plan)
	if err != nil {
		return nil, err
	}
	s.emit(obs.Event{Phase: obs.PhaseSetup, Kind: obs.KindProgress, Network: plan.Network,
		Message: "genesis and config written", Fields: map[string]any{"path": plan.GenesisPath}})
	return specs, nil
}

// launcher builds the LocalLauncher backing this setup.
func (s LocalSetup) launcher() LocalLauncher {
	return LocalLauncher{
		Plugin: s.Plugin, Binary: s.Binary, KeysDir: s.KeysDir,
		Driver: s.Driver, Files: s.Files,
	}
}

// attachGenesis builds the network genesis from the preset — sized to the plan's
// validator count, with the config's overrides/overlay applied — and sets
// plan.Genesis so the launcher can materialize it and init datadirs from it.
func (s LocalSetup) attachGenesis(ctx context.Context, plan *driver.Plan) error {
	gen, err := BuildGenesis(ctx, s.Plugin, GenesisRequest{Validators: planValidatorCount(*plan)}, GenesisConfig{
		KeysDir:         s.KeysDir,
		Binary:          s.Binary,
		ConfigOverrides: genesisConfigOverrides(s.Config),
		Overlay:         []byte(s.Config.String("genesis.overlay", "")),
	})
	if err != nil {
		return err
	}
	plan.Genesis = gen.Genesis
	return nil
}

// planValidatorCount is the number of validator-role nodes in plan: the genesis
// validator set must match exactly the validators being launched (both are the
// first N preset entries), so a topology that places fewer validators than the
// preset holds still gets a consistent genesis.
func planValidatorCount(plan driver.Plan) int {
	n := 0
	for _, spec := range plan.Nodes {
		if node.Is(spec.Role, node.RoleBP) {
			n++
		}
	}
	return n
}

// genesisConfigOverrides collects the genesis config overrides (config keys under
// overridePrefix, e.g. "genesis.overrides.bohoBlock=10") into a bare-key → value
// map for the genesis source. It returns nil when none are set.
func genesisConfigOverrides(cfg config.Values) map[string]string {
	var ov map[string]string
	for k, v := range cfg {
		if suffix, ok := strings.CutPrefix(k, overridePrefix); ok && suffix != "" {
			if ov == nil {
				ov = map[string]string{}
			}
			ov[suffix] = v
		}
	}
	return ov
}
