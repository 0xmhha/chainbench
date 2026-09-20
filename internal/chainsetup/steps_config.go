package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/process"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The config step: each node's TOML, and the argv its launch will use.
//
// Both are rendered from the same resolved values, so a node cannot be
// configured one way and launched another.

func (w *Workspace) Config(ctx context.Context) (string, error) {
	p, err := w.plugin()
	if err != nil {
		return "", err
	}
	if err := w.require("config"); err != nil {
		return "", err
	}
	preset, placed, peering, pubkey, err := w.peerPlan(p)
	if err != nil {
		return "", fmt.Errorf("chainsetup: config: %w", err)
	}
	// Recomputed for this config revision: a re-run reflects the current
	// overrides, not a stale record.
	w.state.ConfigProvenance = w.state.ConfigProvenance[:0]
	overridden := 0
	for _, ns := range w.state.Nodes {
		// Each node's config is rendered from ITS chain. A network of one build
		// resolves to the composition's for every node, which is what this was.
		np, perr := w.pluginFor(ns)
		if perr != nil {
			return "", perr
		}
		prov, err := w.writeNodeConfig(ctx, np, preset, placed, peering, pubkey, ns, "")
		if err != nil {
			return "", err
		}
		w.addConfigProvenance(prov)
		if len(prov.Overrides) > 0 {
			overridden++
		}
	}
	detail := fmt.Sprintf("%d config(s) under %s", len(w.state.Nodes), w.state.Target.DataRoot)
	if overridden > 0 {
		detail += fmt.Sprintf(", %d with overrides", overridden)
	}
	w.markStep("config", detail)
	return detail, nil
}

// writeNodeConfig renders one node's config with its overrides, writes it,
// verifies the readback checksum, and returns its provenance. The Config step
// and a mid-test config swap share it, so both produce the same config and the
// same provenance record. purpose, when set, names the swap's config fixture
// (config-<purpose>); the initial compose passes "".
func (w *Workspace) writeNodeConfig(ctx context.Context, p registry.ChainPlugin, preset keyring.Preset, placed *node.Map, peering node.Peering, pubkey func(int) (string, bool), ns node.Record, purpose string) (ConfigProvenance, error) {
	t, err := w.machineFor(ns)
	if err != nil {
		return ConfigProvenance{}, err
	}
	toml, err := w.nodeConfigBytes(ctx, p, preset, placed, peering, pubkey, ns)
	if err != nil {
		return ConfigProvenance{}, err
	}
	// A node that names its own config file applies no overrides: the file is
	// the whole config.
	var overrides []string
	if ns.Config == "" {
		overrides = w.configOverridesFor(node.Role(ns.Role), ns.Index)
	}
	return w.writeConfigFile(ctx, t, ns, toml, purpose, overrides)
}

// nodeConfigBytes renders what a node's config file would hold, without writing
// anything.
//
// Rendering is separated from writing so a caller can ask what a composition
// WOULD produce before it touches the target. reuse-if-matching needs exactly
// that: it has to decide whether the inputs changed, and deciding after the
// write means a refusal that has already replaced the running network's files.
//
// A node that names its own config file uses it verbatim — the file is the
// whole config, so nothing is rendered and no override applies to it.
func (w *Workspace) nodeConfigBytes(ctx context.Context, p registry.ChainPlugin, preset keyring.Preset, placed *node.Map, peering node.Peering, pubkey func(int) (string, bool), ns node.Record) ([]byte, error) {
	if ns.Config != "" {
		toml, rerr := w.readInputRef(ctx, ns, ns.Config, resource.PurposeConfigs)
		if rerr != nil {
			return nil, fmt.Errorf("chainsetup: config: node%d: read pinned config %s: %w", ns.Index, ns.Config, rerr)
		}
		return toml, nil
	}
	staticNodes, err := node.PeerList(placed, peering, ns.NodeLabel(), pubkey)
	if err != nil {
		return nil, fmt.Errorf("chainsetup: config: node%d peers: %w", ns.Index, err)
	}
	spec := process.NodeConfig(p, preset, process.SpecOf(ns), w.keysBase(), staticNodes)
	if err := w.applyConfigOverrides(&spec, node.Role(ns.Role), ns.Index); err != nil {
		return nil, fmt.Errorf("chainsetup: config: node%d: %w", ns.Index, err)
	}
	// A node whose build reads its genesis from the config carries it here. The
	// file was written by the genesis step beside the network's genesis, and is
	// read back rather than rebuilt so the config holds exactly what that step
	// produced.
	if path := w.genesisConfigFor(ns); path != "" {
		t, terr := w.machineFor(ns)
		if terr != nil {
			return nil, terr
		}
		gen, rerr := t.Files.Read(ctx, path)
		if rerr != nil {
			return nil, fmt.Errorf("chainsetup: config: node%d: read the genesis its config carries (%s): %w", ns.Index, path, rerr)
		}
		spec.Genesis = gen
	}
	return nodeconfig.TOML(spec), nil
}

// writeConfigFile writes one node's config to its target and reads it back:
// the config on the target must hash to what was written, so a truncated or
// clobbered write is caught here rather than at node boot. The checksum runs
// on the target (a remote store runs sha256sum), so it does not download the
// file back. overrides is nil for a pinned config file — it applied none.
func (w *Workspace) writeConfigFile(ctx context.Context, t *resource.Access, ns node.Record, toml []byte, purpose string, overrides []string) (ConfigProvenance, error) {
	w.recordInput(ns.ConfigPath, toml)
	if err := t.Files.Write(ctx, ns.ConfigPath, toml, 0o644); err != nil {
		return ConfigProvenance{}, fmt.Errorf("chainsetup: config: node%d: %w", ns.Index, err)
	}
	want := filestore.Hash(toml)
	got, err := t.Files.Checksum(ctx, ns.ConfigPath)
	if err != nil {
		return ConfigProvenance{}, fmt.Errorf("chainsetup: config: node%d readback: %w", ns.Index, err)
	}
	if got != want {
		return ConfigProvenance{}, fmt.Errorf("chainsetup: config: node%d config did not read back intact (wrote %s, target has %s)", ns.Index, want, got)
	}
	prov := ConfigProvenance{Node: ns.Index, Overrides: overrides, Checksum: want}
	if purpose != "" {
		prov.Fixture = "config-" + purpose
	}
	return prov, nil
}

// LaunchOpts assembles each node's launch argv through nodeconfig.Argv —
// the single argv renderer — and records it in the node table, so `start`
// launches exactly what this step showed. Each node's overrides come from the
// scopes recorded on the workspace (all, then its role, then the node), so one
// argv renderer serves a network of mixed launch flags.
func (w *Workspace) LaunchOpts() (string, error) {
	p, err := w.plugin()
	if err != nil {
		return "", err
	}
	if err := w.require("build"); err != nil {
		return "", err
	}
	preset, placed, peering, pubkey, err := w.peerPlan(p)
	if err != nil {
		return "", fmt.Errorf("chainsetup: launchopts: %w", err)
	}
	scoped := false
	for i, ns := range w.state.Nodes {
		overrides, err := ParseOverrides(w.launchOverridesFor(ns.Role, ns.Index))
		if err != nil {
			return "", err
		}
		if len(overrides) > 0 {
			scoped = true
		}
		staticNodes, err := node.PeerList(placed, peering, ns.NodeLabel(), pubkey)
		if err != nil {
			return "", fmt.Errorf("chainsetup: launchopts: node%d peers: %w", ns.Index, err)
		}
		args, err := nodeconfig.Argv(process.NodeConfig(p, preset, process.SpecOf(ns), w.keysBase(), staticNodes), overrides...)
		if err != nil {
			return "", fmt.Errorf("chainsetup: launchopts: node%d: %w", ns.Index, err)
		}
		w.state.Nodes[i].Args = args
	}
	detail := fmt.Sprintf("%d argv(s) assembled", len(w.state.Nodes))
	if scoped {
		detail += ", with launch overrides"
	}
	w.markStep("build", detail)
	return detail, nil
}

// Provision materializes the shared launch inputs on the target with
// upload-if-absent semantics: the genesis (as built by the genesis step) and
// the per-node configs. Re-running it reuses what already exists.
