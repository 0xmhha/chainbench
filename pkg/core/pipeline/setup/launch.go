package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
func provision(plan *Plan, plugin registry.ChainPlugin, cfg config.Values, preset keys.Preset, keyBase string) error {
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
	// Apply any genesis config overrides (e.g. a delayed fork:
	// genesis.overrides.bohoBlock=10 moves Boho off genesis). Validate the fork
	// ordering of the result so a bad delayed-fork config fails at setup, not at
	// node boot.
	if ov := configOverrides(cfg); len(ov) > 0 {
		gen, err = genesis.ApplyConfigOverrides(gen, ov)
		if err != nil {
			return err
		}
		if err := genesis.ValidateForks(gen); err != nil {
			return fmt.Errorf("setup: genesis overrides: %w", err)
		}
	}
	// A genesis overlay (from `setup --genesis-overlay`, carried as a JSON string
	// in config) deep-merges fragments — e.g. extra alloc accounts with Extra bits
	// — into the built genesis. Re-validate fork ordering after the merge.
	if overlay := cfg.String("genesis.overlay", ""); overlay != "" {
		gen, err = genesis.MergeOverride(gen, []byte(overlay))
		if err != nil {
			return err
		}
		if err := genesis.ValidateForks(gen); err != nil {
			return fmt.Errorf("setup: genesis overlay: %w", err)
		}
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
			KeystoreDir:   filepath.Join(keyBase, fmt.Sprintf("node%d", spec.Index), "keystore"),
			RPCNamespace:  ns,
			SyncMode:      effectiveSyncMode(cfg, spec),
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
	ns, _, err := LaunchWithSpecs(ctx, opts)
	return ns, err
}

// LaunchWithSpecs is Launch that also returns the fully-armed node specs (launch
// args, binary, datadir, config) so the caller can persist them
// (state.SaveNodeSpecs) and later relaunch a single node with RelaunchNode. The
// returned NodeSet is what Launch returns; the specs align with ns.Nodes by
// index.
func LaunchWithSpecs(ctx context.Context, opts LaunchOptions) (node.NodeSet, []driver.NodeSpec, error) {
	if opts.Binary == "" {
		return node.NodeSet{}, nil, fmt.Errorf("setup: launch needs a resolved binary path")
	}
	plan, err := BuildPlan(opts.Config, opts.Plugin, opts.DataRoot)
	if err != nil {
		return node.NodeSet{}, nil, err
	}
	preset, err := keys.LoadPreset(opts.KeysDir)
	if err != nil {
		return node.NodeSet{}, nil, err
	}
	keysAbs, err := filepath.Abs(opts.KeysDir)
	if err != nil {
		return node.NodeSet{}, nil, err
	}
	d := opts.Driver
	if d == nil {
		d = driver.NewLocalDriver()
	}

	// Identity files live at keysAbs locally; a remote driver needs them on its
	// own host, so they are shipped to a keyBase under the (remote) data root and
	// the launch args + config keystore point there. keyBase == keysAbs for a
	// local launch, so local provisioning is unchanged.
	keyBase := keysAbs
	fp, remoteFiles := d.(driver.FileProvisioner)
	if remoteFiles {
		keyBase = filepath.Join(plan.DataRoot, "keys")
	}

	if err := provision(&plan, opts.Plugin, opts.Config, preset, keyBase); err != nil {
		return node.NodeSet{}, nil, err
	}
	if remoteFiles {
		if err := shipIdentities(ctx, fp, keysAbs, keyBase, plan.Nodes); err != nil {
			return node.NodeSet{}, nil, err
		}
	}
	initer, canInit := d.(driver.Initializer)

	// Install each node's preset identity: the devp2p nodekey so its enode
	// matches the static-node list (peering), and — for validators — the account
	// to unlock and seal with (a random key is otherwise "unauthorized").
	for i := range plan.Nodes {
		spec := &plan.Nodes[i]
		nodeDir := filepath.Join(keyBase, fmt.Sprintf("node%d", spec.Index))
		spec.Args = append(spec.Args, "--nodekey", filepath.Join(nodeDir, "nodekey"))
		if spec.Role == node.RoleValidator {
			if nk := nodeKeyFor(preset, spec.Index); nk != nil {
				spec.Args = append(spec.Args,
					"--unlock", nk.Address,
					"--password", filepath.Join(keyBase, "password"),
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
				return node.NodeSet{}, nil, err
			}
		} else if err := driver.InitDatadir(ctx, opts.Binary, spec.DataDir, plan.GenesisPath); err != nil {
			return node.NodeSet{}, nil, err
		}
	}
	plan.Genesis = nil // already written by provision + placed by init
	ns, err := Run(ctx, plan, d, opts.Bus)
	// plan.Nodes now carries the fully-armed specs (identity args, binary); the
	// caller persists them so a single node can be relaunched after a stop.
	return ns, plan.Nodes, err
}

// shipIdentities copies each node's preset identity files — the devp2p nodekey,
// the validator keystore, and the shared password — from the local keysAbs to
// keyBase on the driver's host via the FileProvisioner, so a remote launch finds
// them at the paths its launch args and config keystore reference.
func shipIdentities(ctx context.Context, fp driver.FileProvisioner, keysAbs, keyBase string, nodes []driver.NodeSpec) error {
	if pw, err := os.ReadFile(filepath.Join(keysAbs, "password")); err == nil {
		if err := fp.ProvisionFile(ctx, filepath.Join(keyBase, "password"), pw, 0o600); err != nil {
			return err
		}
	}
	for _, spec := range nodes {
		src := filepath.Join(keysAbs, fmt.Sprintf("node%d", spec.Index))
		dst := filepath.Join(keyBase, fmt.Sprintf("node%d", spec.Index))
		if nk, err := os.ReadFile(filepath.Join(src, "nodekey")); err == nil {
			if err := fp.ProvisionFile(ctx, filepath.Join(dst, "nodekey"), nk, 0o600); err != nil {
				return err
			}
		}
		entries, err := os.ReadDir(filepath.Join(src, "keystore"))
		if err != nil {
			continue // no keystore (e.g. an endpoint node)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			b, err := os.ReadFile(filepath.Join(src, "keystore", e.Name()))
			if err != nil {
				return err
			}
			if err := fp.ProvisionFile(ctx, filepath.Join(dst, "keystore", e.Name()), b, 0o600); err != nil {
				return err
			}
		}
	}
	return nil
}

// overridePrefix is the config namespace whose keys are merged into the genesis
// `config` object (e.g. "genesis.overrides.bohoBlock=10"). It mirrors the bash
// profile schema (genesis.overrides.*), so a delayed-fork profile flattens
// straight into this namespace.
const overridePrefix = "genesis.overrides."

// configOverrides collects the genesis config overrides (keys under
// overridePrefix) into a map of bare config key → value for
// genesis.ApplyConfigOverrides. It returns nil when none are set.
func configOverrides(cfg config.Values) map[string]string {
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

// effectiveSyncMode is the node's sync mode: an explicit per-node spec.SyncMode
// (from a topology config) wins; otherwise it falls back to the role-based
// default.
func effectiveSyncMode(cfg config.Values, spec *driver.NodeSpec) string {
	if spec.SyncMode != "" {
		return spec.SyncMode
	}
	return syncModeFor(cfg, spec.Role)
}

// syncModeFor returns the geth sync mode for a node's role. Validators always
// use "full" — they must hold full state to seal blocks — while endpoints may be
// switched to "snap" (config nodes.endpoint_syncmode) so a large-gap re-sync
// exercises the snap-sync path (regression a1-03).
func syncModeFor(cfg config.Values, role node.Role) string {
	if role == node.RoleEndpoint {
		return cfg.String("nodes.endpoint_syncmode", "full")
	}
	return "full"
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
