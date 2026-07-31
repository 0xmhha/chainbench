// Package setup is the first pipeline phase (requirement #8): it turns resolved
// config + a chain plugin into a running (or provisioned) NodeSet through three
// steps — plan (config -> node specs), provision (env build: genesis, config,
// datadir), and launch (driver). It is chain-agnostic; genesis/config content
// comes from the consensus family and chain plugin
// (docs/CHAINBENCH_GO_REDESIGN.md §3, §8).
package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/pkg/core/config"
	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/nodeconfig"
	"github.com/0xmhha/chainbench/pkg/core/obs"
	"github.com/0xmhha/chainbench/pkg/core/registry"
	"github.com/0xmhha/chainbench/pkg/core/topology"
)

// Plan is the fully-resolved setup description: what genesis to write and which
// nodes to provision/launch. Building a Plan performs no I/O, so it is unit
// testable and inspectable before anything runs.
type Plan struct {
	Chain        string
	Network      string
	DataRoot     string
	GenesisPath  string
	Genesis      []byte // genesis.json bytes (supplied by the consensus family)
	Capabilities []string
	Nodes        []driver.NodeSpec
}

// placement is one node's resolved role + sync mode, in launch order.
type placement struct {
	role     node.Role
	syncMode string // "" -> role-based default
}

// BuildPlan plans a local setup from resolved config and a chain plugin. Node 1..V
// are validators, V+1..V+E are endpoints; each node gets ports offset by its
// zero-based index (requirement #6). Genesis bytes are attached separately (the
// caller obtains them from the consensus family) so planning stays pure.
func BuildPlan(cfg config.Values, plugin registry.ChainPlugin, dataRoot string) (Plan, error) {
	return BuildPlanWithTopology(cfg, plugin, dataRoot, nil)
}

// BuildPlanWithTopology is BuildPlan with an optional explicit per-node topology.
// When topo is nil the positional "nodes.validators + nodes.endpoints" counts
// drive the layout; when it is set, each node's role and sync mode come from the
// topology (its Nodes must already be Validate()d). Node index i (1-based) is the
// i-th entry in launch order.
func BuildPlanWithTopology(cfg config.Values, plugin registry.ChainPlugin, dataRoot string, topo *topology.Topology) (Plan, error) {
	var placements []placement
	if topo != nil {
		for _, n := range topo.Sorted() {
			placements = append(placements, placement{role: n.NodeRole(), syncMode: n.EffectiveSyncMode()})
		}
	} else {
		validators := cfg.Int("nodes.validators", 0)
		endpoints := cfg.Int("nodes.endpoints", 0)
		for i := 0; i < validators; i++ {
			placements = append(placements, placement{role: node.RoleValidator})
		}
		for i := 0; i < endpoints; i++ {
			placements = append(placements, placement{role: node.RoleEndpoint})
		}
	}
	total := len(placements)
	if total < 1 {
		return Plan{}, fmt.Errorf("setup: need at least one node")
	}

	m := plugin.Manifest()
	fam := plugin.Family()
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
		ports := node.Offset(base, i-1)
		dataDir := filepath.Join(dataRoot, fmt.Sprintf("node%d", i))
		configPath := filepath.Join(dataRoot, fmt.Sprintf("config_node%d.toml", i))
		nodes = append(nodes, driver.NodeSpec{
			Index:      i,
			Role:       p.role,
			SyncMode:   p.syncMode,
			Host:       host,
			Binary:     m.Binary,
			DataDir:    dataDir,
			ConfigPath: configPath,
			LogPath:    filepath.Join(dataRoot, "logs", fmt.Sprintf("node%d.log", i)),
			Ports:      ports,
			Args:       nodeconfig.LaunchArgs(dataDir, configPath, ports, fam.StartFlags(p.role)),
		})
	}

	// A delayed Boho activation (genesis.overrides.bohoBlock=N>0) is advertised as
	// a capability so the fork-transition test cases gate on it and skip on a
	// normal (Boho-at-genesis) network. A genesis overlay may declare further
	// capabilities (config "genesis.capabilities", comma-separated) — e.g. an
	// account-extra overlay advertises "account-extra".
	caps := append([]string(nil), m.Capabilities...)
	// Launched nodes serve a WebSocket endpoint (--ws), so advertise "ws" for the
	// eth_subscribe cases; attached networks (no known WS port) do not.
	caps = append(caps, "ws")
	if cfg.Int(overridePrefix+"bohoBlock", 0) > 0 {
		caps = append(caps, "delayed-boho")
	}
	for _, c := range strings.Split(cfg.String("genesis.capabilities", ""), ",") {
		if c = strings.TrimSpace(c); c != "" {
			caps = append(caps, c)
		}
	}

	return Plan{
		Chain:        m.ID,
		Network:      "local",
		DataRoot:     dataRoot,
		GenesisPath:  filepath.Join(dataRoot, "genesis.json"),
		Capabilities: caps,
		Nodes:        nodes,
	}, nil
}

// Run executes a Plan against a Driver: it writes genesis, then provisions and
// launches each node, emitting obs events. It returns the resulting NodeSet.
// bus may be nil.
func Run(ctx context.Context, plan Plan, d driver.Driver, bus *obs.Bus) (node.NodeSet, error) {
	ns := node.NodeSet{Chain: plan.Chain, Network: plan.Network, Capabilities: plan.Capabilities}

	if len(plan.Genesis) > 0 {
		if err := os.MkdirAll(plan.DataRoot, 0o755); err != nil {
			return ns, fmt.Errorf("setup: mkdir data root: %w", err)
		}
		if err := os.WriteFile(plan.GenesisPath, plan.Genesis, 0o644); err != nil {
			return ns, fmt.Errorf("setup: write genesis: %w", err)
		}
		emit(bus, obs.Event{Phase: obs.PhaseSetup, Kind: obs.KindProgress, Network: plan.Network,
			Message: "genesis written", Fields: map[string]any{"path": plan.GenesisPath}})
	}

	for _, spec := range plan.Nodes {
		if err := d.Provision(ctx, spec); err != nil {
			emit(bus, obs.Event{Phase: obs.PhaseSetup, Kind: obs.KindError, Network: plan.Network,
				Node: spec.Index, Message: "provision failed", Fields: map[string]any{"error": err.Error()}})
			return ns, fmt.Errorf("setup: provision node%d: %w", spec.Index, err)
		}
		h, err := d.Launch(ctx, spec)
		if err != nil {
			emit(bus, obs.Event{Phase: obs.PhaseSetup, Kind: obs.KindError, Network: plan.Network,
				Node: spec.Index, Message: "launch failed", Fields: map[string]any{"error": err.Error()}})
			return ns, fmt.Errorf("setup: launch node%d: %w", spec.Index, err)
		}
		ns.Nodes = append(ns.Nodes, node.Node{
			Index:  spec.Index,
			Role:   spec.Role,
			Host:   spec.Host,
			RPCURL: fmt.Sprintf("http://%s:%d", spec.Host, spec.Ports.HTTP),
			Ports:  spec.Ports,
			PID:    h.PID,
		})
		emit(bus, obs.Event{Phase: obs.PhaseSetup, Kind: obs.KindProgress, Network: plan.Network,
			Node: spec.Index, Message: "node launched",
			Fields: map[string]any{"role": string(spec.Role), "pid": h.PID, "http": spec.Ports.HTTP}})
	}

	emit(bus, obs.Event{Phase: obs.PhaseSetup, Kind: obs.KindResult, Network: plan.Network,
		Message: "setup complete", Fields: map[string]any{"nodes": len(ns.Nodes)}})
	return ns, nil
}

func emit(bus *obs.Bus, e obs.Event) {
	if bus != nil {
		bus.Publish(e)
	}
}
