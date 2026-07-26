package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/pkg/core/config"
	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/genesis"
	"github.com/0xmhha/chainbench/pkg/core/keys"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/nodeconfig"
	"github.com/0xmhha/chainbench/pkg/core/obs"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

// LaunchOptions configures a local provision-and-launch. Binary is the resolved
// node executable path; KeysDir is the preset key set. It is the shared core of
// `chainbench setup --launch` and the start MCP tool, so both provision, install
// identities, initialize datadirs, and launch identically.
type LaunchOptions struct {
	Plugin   registry.ChainPlugin
	Config   config.Values
	DataRoot string
	Binary   string
	KeysDir  string
	Bus      *obs.Bus // optional; events are dropped when nil
	// Driver launches the nodes; nil defaults to the local driver. Inject a
	// driver.NewRemoteDriver(...) to provision the per-node config and launch
	// over SSH. (Genesis writing and datadir init remain local for now — remote
	// genesis-shipping and remote `init` are a follow-up.)
	Driver driver.Driver
}

// Provision writes the genesis and per-node config files from the preset keys,
// the on-disk artifacts a launch (or an external launcher) then boots from. It
// does not start any process. The per-node config is written through the local
// driver (the same path Launch routes through a possibly-remote driver), not by
// a direct file write, so provisioning stays behind the Driver seam.
func Provision(ctx context.Context, plan Plan, plugin registry.ChainPlugin, cfg config.Values, keysDir string) error {
	preset, err := keys.LoadPreset(keysDir)
	if err != nil {
		return err
	}
	keysAbs, err := filepath.Abs(keysDir)
	if err != nil {
		return err
	}
	if err := provision(&plan, plugin, cfg, preset, keysAbs); err != nil {
		return err
	}
	ld := driver.NewLocalDriver()
	for _, spec := range plan.Nodes {
		if err := ld.Provision(ctx, spec); err != nil {
			return err
		}
	}
	return nil
}

// provision is the shared body used by Provision and Launch (which already holds
// the loaded preset). It writes the network-wide genesis file and fills each
// node spec's ConfigContent with the rendered TOML; the config bytes are written
// to their destination by the driver's Provision (local or remote), never here,
// so a remote launch ships the config over its transport.
func provision(plan *Plan, plugin registry.ChainPlugin, cfg config.Values, preset keys.Preset, keysAbs string) error {
	sub := preset.Take(cfg.Int("nodes.validators", len(preset.Validators)))
	gen, err := genesis.Build(plugin, genesis.Inputs{
		Validators: sub.Validators,
		BLSKeys:    sub.BLSKeys,
		ExtraData:  sub.ExtraData,
		Members:    sub.Members,
		Alloc:      sub.Alloc,
	})
	if err != nil {
		return err
	}
	if err := os.MkdirAll(plan.DataRoot, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(plan.GenesisPath, gen, 0o644); err != nil {
		return err
	}
	plan.Genesis = gen // kept so the datadir init can place it (local or remote)

	// static-nodes: every preset node's enode at its planned p2p port.
	var staticNodes []string
	for _, spec := range plan.Nodes {
		if nk := nodeKeyFor(preset, spec.Index); nk != nil {
			staticNodes = append(staticNodes, nodeconfig.Enode(nk.PublicKey, spec.Host, spec.Ports.P2P))
		}
	}
	ns := plugin.Manifest().Consensus.RPCNamespace
	recommit := plugin.Manifest().MinerRecommit
	for i := range plan.Nodes {
		spec := &plan.Nodes[i]
		spec.ConfigContent = nodeconfig.Generate(nodeconfig.Params{
			Role:          spec.Role,
			Ports:         spec.Ports,
			KeystoreDir:   filepath.Join(keysAbs, fmt.Sprintf("node%d", spec.Index), "keystore"),
			RPCNamespace:  ns,
			MinerRecommit: recommit,
			StaticNodes:   staticNodes,
		})
	}
	return nil
}

// Launch provisions, installs each node's preset identity, initializes the
// datadirs with the binary, and launches the network, returning the NodeSet. The
// caller resolves opts.Binary and persists the returned NodeSet (state.SaveNodeSet).
func Launch(ctx context.Context, opts LaunchOptions) (node.NodeSet, error) {
	if opts.Binary == "" {
		return node.NodeSet{}, fmt.Errorf("setup: launch needs a resolved binary path")
	}
	plan, err := BuildPlan(opts.Config, opts.Plugin, opts.DataRoot)
	if err != nil {
		return node.NodeSet{}, err
	}
	preset, err := keys.LoadPreset(opts.KeysDir)
	if err != nil {
		return node.NodeSet{}, err
	}
	keysAbs, err := filepath.Abs(opts.KeysDir)
	if err != nil {
		return node.NodeSet{}, err
	}
	if err := provision(&plan, opts.Plugin, opts.Config, preset, keysAbs); err != nil {
		return node.NodeSet{}, err
	}

	d := opts.Driver
	if d == nil {
		d = driver.NewLocalDriver()
	}
	initer, canInit := d.(driver.Initializer)

	// Install each node's preset identity: the devp2p nodekey so its enode
	// matches the static-node list (peering), and — for validators — the account
	// to unlock and seal with (a random key is otherwise "unauthorized").
	for i := range plan.Nodes {
		spec := &plan.Nodes[i]
		nodeDir := filepath.Join(keysAbs, fmt.Sprintf("node%d", spec.Index))
		spec.Args = append(spec.Args, "--nodekey", filepath.Join(nodeDir, "nodekey"))
		if spec.Role == node.RoleValidator {
			if nk := nodeKeyFor(preset, spec.Index); nk != nil {
				spec.Args = append(spec.Args,
					"--unlock", nk.Address,
					"--password", filepath.Join(keysAbs, "password"),
					"--miner.etherbase", nk.Address,
				)
			}
		}
		spec.Binary = opts.Binary
		// Init the datadir through the driver when it supports it (so a remote
		// driver ships the genesis and runs init on its host); otherwise fall
		// back to the local init from the on-disk genesis.
		if canInit {
			if err := initer.InitDatadir(ctx, *spec, plan.Genesis); err != nil {
				return node.NodeSet{}, err
			}
		} else if err := driver.InitDatadir(ctx, opts.Binary, spec.DataDir, plan.GenesisPath); err != nil {
			return node.NodeSet{}, err
		}
	}
	plan.Genesis = nil // already written by provision + placed by init
	return Run(ctx, plan, d, opts.Bus)
}

// nodeKeyFor returns the preset node key for a 1-based node index, or nil.
func nodeKeyFor(p keys.Preset, index int) *keys.NodeKey {
	for i := range p.Nodes {
		if p.Nodes[i].Index == index {
			return &p.Nodes[i]
		}
	}
	return nil
}
