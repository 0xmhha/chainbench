package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/core/bringup"
	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/state"
	"github.com/0xmhha/chainbench/internal/core/topology"
)

// Bring-up use cases for the setup stack: resolve a chain and a layout into a
// plan, then either write its artifacts or launch it. The CLI's `setup` command
// and the start MCP tool are two renderings of these three functions, so they
// cannot drift (worklist T7.11).

// topologyFile is the copy of the resolved topology kept in the data root, so a
// running network's layout is inspectable from its own directory.
const topologyFile = "topology.yaml"

// NetworkSpecIn describes the network to plan: which chain, how many nodes of
// each role, and what to override. It is the shared input of NetworkPlan,
// NetworkProvision, and NetworkLaunch.
type NetworkSpecIn struct {
	// Chain is the embedded chain id. Ignored when ManifestPath is set, and
	// overridden by a topology file's own chain when the caller did not ask
	// for a specific one (ChainExplicit).
	Chain string
	// ChainExplicit marks Chain as a deliberate choice, so a topology file's
	// chain does not silently win over it.
	ChainExplicit bool
	// ManifestPath selects an external, project-supplied chain manifest (the
	// hybrid model) instead of an embedded chain. TemplatePath is its genesis
	// template.
	ManifestPath string
	TemplatePath string
	// TopologyPath is an optional per-node layout YAML (role, sync mode,
	// bootnode). It replaces the positional Validators/Endpoints counts.
	TopologyPath string
	// DataDir is the network's data root.
	DataDir string
	// Validators and Endpoints override the configured counts. Nil leaves the
	// configured value: zero endpoints is a meaningful request, so "not given"
	// cannot be encoded as 0.
	Validators *int
	Endpoints  *int
	// Set carries flat config overrides as key=value, e.g.
	// "genesis.overrides.bohoBlock=10" for a delayed-fork network.
	Set []string
	// GenesisOverlayPath is a JSON overlay file {capabilities, genesis}: the
	// genesis fragment is deep-merged into the built genesis and the
	// capabilities are advertised on the node set, so overlay-gated cases run.
	GenesisOverlayPath string
}

// NetworkPlanOut is the resolved bring-up description. The plan performs no
// I/O, so a surface can print it before anything runs.
type NetworkPlanOut struct {
	// Plugin is the resolved chain (embedded or external manifest).
	Plugin registry.ChainPlugin
	// Config is the resolved configuration the plan was built from.
	Config config.Values
	// Plan is the per-node layout.
	Plan driver.Plan
	// BootnodeIndex is the topology's bootnode, or 0 when there is none.
	BootnodeIndex int
	// DataRoot is the cleaned data root the plan targets.
	DataRoot string
}

// NetworkPlan resolves the chain, the layout, and the configuration into a
// plan. It touches the filesystem only to read the topology and overlay files
// the caller named.
func NetworkPlan(_ context.Context, _ Deps, in NetworkSpecIn) (NetworkPlanOut, error) {
	chain, topo, err := resolveTopology(in)
	if err != nil {
		return NetworkPlanOut{}, err
	}
	plugin, err := ResolveChain(chain, in.ManifestPath, in.TemplatePath)
	if err != nil {
		return NetworkPlanOut{}, err
	}
	override, err := overrides(in)
	if err != nil {
		return NetworkPlanOut{}, err
	}
	cfg := config.Resolve(nil, override)
	root := filepath.Clean(in.DataDir)
	plan, err := bringup.BuildPlanWithTopology(cfg, plugin, root, topo)
	if err != nil {
		return NetworkPlanOut{}, err
	}
	out := NetworkPlanOut{Plugin: plugin, Config: cfg, Plan: plan, DataRoot: root}
	if topo != nil {
		out.BootnodeIndex = topo.BootnodeIndex()
	}
	return out, nil
}

// NetworkProvisionIn asks for the on-disk artifacts of a planned network.
type NetworkProvisionIn struct {
	Spec NetworkSpecIn
	// KeysDir is the preset key set the genesis and identities come from.
	KeysDir string
}

// NetworkProvisionOut reports what was written.
type NetworkProvisionOut struct {
	Plan NetworkPlanOut
}

// NetworkProvision writes the genesis and per-node configs of a planned
// network. It starts no process, so an external launcher can boot from the
// result.
func NetworkProvision(ctx context.Context, d Deps, in NetworkProvisionIn) (NetworkProvisionOut, error) {
	planned, err := NetworkPlan(ctx, d, in.Spec)
	if err != nil {
		return NetworkProvisionOut{}, err
	}
	if err := bringup.Provision(ctx, planned.Plan, planned.Plugin, planned.Config, in.KeysDir); err != nil {
		return NetworkProvisionOut{}, err
	}
	if err := saveTopology(planned.DataRoot, in.Spec.TopologyPath); err != nil {
		return NetworkProvisionOut{}, err
	}
	return NetworkProvisionOut{Plan: planned}, nil
}

// NetworkLaunchIn asks for a planned network to be brought up.
type NetworkLaunchIn struct {
	Spec NetworkSpecIn
	// KeysDir is the preset key set.
	KeysDir string
	// Binary is the resolved node executable. For a remote launch it is a path
	// on the remote host, so the caller resolves it — this layer never looks it
	// up on PATH.
	Binary string
	// Bus receives bring-up events; nil disables emission.
	Bus *obs.Bus
}

// NetworkLaunchOut is the launched network.
type NetworkLaunchOut struct {
	Plan NetworkPlanOut
	// Nodes is the running node set, also persisted to the data root.
	Nodes node.NodeSet
}

// NetworkLaunch provisions, installs the node identities, initializes the
// datadirs, and starts the network, persisting both the running node set and
// the armed launch specs. The specs are what lets a single node be stopped and
// brought back later without replanning the network.
func NetworkLaunch(ctx context.Context, d Deps, in NetworkLaunchIn) (NetworkLaunchOut, error) {
	if in.Binary == "" {
		return NetworkLaunchOut{}, errors.New("app: launch needs a resolved node binary path")
	}
	planned, err := NetworkPlan(ctx, d, in.Spec)
	if err != nil {
		return NetworkLaunchOut{}, err
	}
	// A nil Deps.Driver means the local host; setup treats a nil driver the
	// same way, so the two defaults agree.
	var dr driver.Driver
	if d.Driver != nil {
		if dr, err = d.Driver(); err != nil {
			return NetworkLaunchOut{}, err
		}
	}
	ns, specs, err := bringup.LaunchWithSpecs(ctx, bringup.LaunchOptions{
		Plugin: planned.Plugin, Config: planned.Config, DataRoot: planned.DataRoot,
		Binary: in.Binary, KeysDir: in.KeysDir, Bus: in.Bus, Driver: dr,
	})
	if err != nil {
		return NetworkLaunchOut{}, err
	}
	if err := state.SaveNodeSet(planned.DataRoot, ns); err != nil {
		return NetworkLaunchOut{}, err
	}
	if err := state.SaveNodeSpecs(planned.DataRoot, specs); err != nil {
		return NetworkLaunchOut{}, err
	}
	if err := saveTopology(planned.DataRoot, in.Spec.TopologyPath); err != nil {
		return NetworkLaunchOut{}, err
	}
	return NetworkLaunchOut{Plan: planned, Nodes: ns}, nil
}

// resolveTopology loads the optional topology file and reports which chain the
// network is for. A topology names its own chain, which wins only when the
// caller did not ask for a specific one.
func resolveTopology(in NetworkSpecIn) (string, *topology.Topology, error) {
	if in.TopologyPath == "" {
		return in.Chain, nil, nil
	}
	loaded, err := topology.Load(in.TopologyPath)
	if err != nil {
		return "", nil, err
	}
	chain := in.Chain
	if !in.ChainExplicit && loaded.Chain != "" {
		chain = loaded.Chain
	}
	return chain, &loaded, nil
}

// ResolveChain returns the chain plugin: an external, project-supplied manifest
// when one is named (the hybrid model), otherwise the embedded chain registered
// for the id. Exported because every surface that acts on a chain resolves it
// the same way; the implementation lives in chains/external, which is the one
// package that knows both halves.
func ResolveChain(chain, manifestPath, templatePath string) (registry.ChainPlugin, error) {
	return external.ResolveChain(chain, manifestPath, templatePath)
}

// overrides folds the node counts, the flat --set pairs, and the genesis
// overlay file into one config override map.
func overrides(in NetworkSpecIn) (config.Values, error) {
	ov := config.Values{}
	if in.Validators != nil {
		ov["nodes.validators"] = strconv.Itoa(*in.Validators)
	}
	if in.Endpoints != nil {
		ov["nodes.endpoints"] = strconv.Itoa(*in.Endpoints)
	}
	for _, kv := range in.Set {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("app: config override expects key=value, got %q", kv)
		}
		ov[k] = v
	}
	if in.GenesisOverlayPath == "" {
		return ov, nil
	}
	raw, err := os.ReadFile(in.GenesisOverlayPath)
	if err != nil {
		return nil, err
	}
	var overlay struct {
		Capabilities []string        `json:"capabilities"`
		Genesis      json.RawMessage `json:"genesis"`
	}
	if err := json.Unmarshal(raw, &overlay); err != nil {
		return nil, fmt.Errorf("app: bad genesis overlay %q: %w", in.GenesisOverlayPath, err)
	}
	if len(overlay.Genesis) > 0 {
		ov["genesis.overlay"] = string(overlay.Genesis)
	}
	if len(overlay.Capabilities) > 0 {
		ov["genesis.capabilities"] = strings.Join(overlay.Capabilities, ",")
	}
	return ov, nil
}

// saveTopology copies the resolved topology into the data root so the running
// network's layout — which node plays which role — is readable from its own
// directory. A no-op when no topology file was used.
func saveTopology(root, topologyPath string) error {
	if topologyPath == "" {
		return nil
	}
	b, err := os.ReadFile(topologyPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, topologyFile), b, 0o644)
}
