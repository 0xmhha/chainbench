package chainsetup

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/topology"
)

// overridePrefix is the config namespace whose keys are merged into the genesis
// `config` object (e.g. "genesis.overrides.bohoBlock=10"), mirroring the bash
// profile schema so a delayed-fork profile flattens straight into it.
const overridePrefix = "genesis.overrides."

// localPlacement is one node's resolved role + sync mode, in launch order.
type localPlacement struct {
	role     node.Role
	syncMode string // "" -> role-based default
}

// BuildLocalPlan builds a local-host driver.Plan from resolved config and a
// chain plugin, with an optional explicit per-node topology. When topo is nil
// the positional "nodes.validators + nodes.endpoints" counts drive the layout;
// when set, each node's role and sync mode come from the topology (its Nodes
// must already be Validate()d). Node index i (1-based) is the i-th entry in
// launch order, its ports offset from the config base by i-1.
//
// It performs no I/O and leaves each spec's Args empty: the launch argv is
// assembled single-sited by LocalLauncher.Arm (launchopt Builder), so the plan
// stays a pure description the CLI can display and the launcher can arm. It is
// the fixed-base-port counterpart to AssemblePlan (which takes allocator ports),
// preserving the CLI's ports.base_* scheme.
func BuildLocalPlan(cfg config.Values, plugin registry.ChainPlugin, dataRoot string, topo *topology.Topology) (driver.Plan, error) {
	var placements []localPlacement
	if topo != nil {
		for _, n := range topo.Sorted() {
			placements = append(placements, localPlacement{role: n.NodeRole(), syncMode: n.EffectiveSyncMode()})
		}
	} else {
		for i := 0; i < cfg.Int("nodes.validators", 0); i++ {
			placements = append(placements, localPlacement{role: node.RoleValidator})
		}
		for i := 0; i < cfg.Int("nodes.endpoints", 0); i++ {
			placements = append(placements, localPlacement{role: node.RoleEndpoint})
		}
	}
	total := len(placements)
	if total < 1 {
		return driver.Plan{}, fmt.Errorf("engine: local plan needs at least one node")
	}

	m := plugin.Manifest()
	host := "127.0.0.1"
	base := node.Endpoints{
		P2P:     cfg.Int("ports.base_p2p", 30301),
		HTTP:    cfg.Int("ports.base_http", 8501),
		WS:      cfg.Int("ports.base_ws", 9501),
		Auth:    cfg.Int("ports.base_auth", 8551),
		Metrics: cfg.Int("ports.base_metrics", 6061),
	}

	nodes := make([]driver.NodeSpec, 0, total)
	for i := 1; i <= total; i++ {
		p := placements[i-1]
		nodes = append(nodes, driver.NodeSpec{
			Index:      i,
			Role:       p.role,
			SyncMode:   p.syncMode,
			Host:       host,
			Binary:     m.Binary,
			DataDir:    filepath.Join(dataRoot, fmt.Sprintf("node%d", i)),
			ConfigPath: filepath.Join(dataRoot, fmt.Sprintf("config_node%d.toml", i)),
			LogPath:    filepath.Join(dataRoot, "logs", fmt.Sprintf("node%d.log", i)),
			Ports:      node.Offset(base, i-1),
		})
	}

	return driver.Plan{
		Chain:        m.ID,
		Network:      "local",
		DataRoot:     dataRoot,
		GenesisPath:  filepath.Join(dataRoot, "genesis.json"),
		Capabilities: localPlanCapabilities(cfg, m),
		Nodes:        nodes,
	}, nil
}

// localPlanCapabilities derives a launched local network's advertised
// capabilities: the manifest's own, plus "ws" (launched nodes serve --ws),
// "delayed-boho" when a delayed Boho activation is configured
// (genesis.overrides.bohoBlock>0, gating the fork-transition cases), and any
// overlay-declared capabilities (config "genesis.capabilities", comma-separated
// — e.g. an account-extra overlay advertising "account-extra").
func localPlanCapabilities(cfg config.Values, m registry.Manifest) []string {
	caps := append([]string(nil), m.Capabilities...)
	caps = append(caps, "ws")
	if cfg.Int(overridePrefix+"bohoBlock", 0) > 0 {
		caps = append(caps, "delayed-boho")
	}
	for _, c := range strings.Split(cfg.String("genesis.capabilities", ""), ",") {
		if c = strings.TrimSpace(c); c != "" {
			caps = append(caps, c)
		}
	}
	return caps
}
