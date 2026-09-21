package chainsetup

import (
	"context"
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/core/blueprint"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The composition verbs, as a surface calls them: keys, allocate, genesis,
// config, launch options.
//
// These carry more than a call because a declaration reaches them here — a
// blueprint to read, an overlay to stage, a peering to resolve. The lifecycle
// Placement bounds shared by the composition steps: a BFT network needs at
// least one validator, and a local port band holds this many nodes.
const (
	minValidators = 1
	portBand      = 100
)

// NetKeysIn selects where node identities come from.
type NetKeysIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	// Source is preset (default) | generate | declared.
	Source string `cb:"keys-source" default:"keyPreset" help:"keyPreset (use the recorded key set) | generate (create a fresh set)"`
	// BlueprintPath is the declaration the keys come from when Source is
	// "declared" (or when a blueprint is given and Source is silent).
	BlueprintPath string
	Nodes         int `cb:"nodes" help:"identities the set must cover (default: the allocated node count)"`
	Validators    int `cb:"validators" help:"identities joining the validator set (generate; 0 = all)"`
}

// NetKeys ensures the workspace's key set exists and covers the node count.
func NetKeys(ctx context.Context, d Deps, in NetKeysIn) (StepOut, error) {
	bp, err := readBlueprint(in.BlueprintPath)
	if err != nil {
		return StepOut{}, err
	}
	source := in.Source
	// A blueprint that carries keys is the source unless the caller asked for
	// another one. Making the operator name it twice would let the two answers
	// disagree, and the composition would take the one they did not mean.
	if source == "" && bp != nil {
		source = "declared"
	}
	return inWorkspace(d, in.DataDir, func(ws *Workspace) (StepOut, error) {
		return ws.Keys(ctx, KeysOpts{Source: source, Blueprint: bp, Nodes: in.Nodes, Validators: in.Validators})
	})
}

// NetAllocateIn sizes the network.
type NetAllocateIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	BPCount int    `cb:"bp" default:"4" help:"bp (block-producing) node count"`
	ENCount int    `cb:"en" help:"en (endpoint, non-producing) node count"`
	PNCount int    `cb:"pn" help:"pn (proxy-tier) node count; a family with no proxy tier refuses it"`
	// Peering is the peer graph ("mesh" default, "proxied").
	Peering string `cb:"peering" help:"peer graph: mesh (default, every node dials every other) | proxied (bp <-> pn <-> en; endpoints never dial a producer)"`
	// EndpointSyncMode switches endpoints off full sync ("snap"/"archive") so a
	// re-sync test can exercise that path. Empty leaves every node on full.
	EndpointSyncMode string `cb:"endpoint-syncmode" help:"sync mode for endpoints (snap|archive); default full"`
	// TopologyPath is a per-node layout YAML (role, sync mode, bootnode). It
	// replaces the counts, which cannot express a per-node choice.
	TopologyPath string `cb:"topology" help:"per-node layout YAML (role/sync-mode/bootnode/binary); overrides --bp/--en/--pn"`
	// BlueprintPath is a network declaration (N1). It is the widest of the
	// three layout sources and wins over both the counts and a topology.
	BlueprintPath string
	// Server selects where the nodes are placed and on what ports, from the
	// operator's server set. Its zero value uses the built-in local plan.
	Server resource.ServerRef
	// Topology, when set, is the inline per-node layout (role/sync/bootnode/
	// binary), the DSL's way to declare what --topology gives a file. It wins
	// over Validators/Endpoints.
	Topology *node.Topology
	// Binaries maps per-node binary names to paths (with a per-node topology).
	Binaries map[string]string
	// BinaryChains names, per binary, the chain that binary runs when it is not
	// the composition's.
	BinaryChains map[string]string
	// AutoSize fills the validator count to the server set (bp: "max"): the
	// network is one node per server, less the proxies and endpoints asked for.
	// It needs a server-set target and is ignored when a topology or blueprint
	// gives the layout explicitly.
	AutoSize bool
}

// NetAllocate builds the node table (roles, paths, deterministic ports).
func NetAllocate(_ context.Context, d Deps, in NetAllocateIn) (StepOut, error) {
	bp, err := readBlueprint(in.BlueprintPath)
	if err != nil {
		return StepOut{}, err
	}
	topo := in.Topology
	if bp != nil && (topo != nil || in.TopologyPath != "") {
		// Both describe the layout, and picking one silently would leave the
		// other's author reading a network that is not theirs.
		return StepOut{}, ofKind(errPlaceTwoLayouts,
			fmt.Errorf("chainsetup: allocate: a blueprint and a topology both describe the layout — give one"))
	}
	if topo == nil && in.TopologyPath != "" {
		loaded, err := node.Load(in.TopologyPath)
		if err != nil {
			return StepOut{}, err
		}
		topo = &loaded
	}
	// Allocation is the one moment two runs can hand out the same slot: each
	// derives the set's inventory from the workspaces it can see, and two
	// runs that look before either has saved both see it free. The set's
	// lock is held from the look to the save — a lock, not a second record
	// (the workspaces stay the only record of what is taken).
	setPath := in.Server.SetPath
	if setPath == "" {
		if ws, err := Open(in.DataDir, d.Clock); err == nil {
			setPath = ws.State().ServerSet
		}
	}
	release, err := acquireSetLock(setPath, d)
	if err != nil {
		return StepOut{}, err
	}
	defer release()
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		// A set the workspace already recorded (chain new --server-set) is the
		// default: --docker and its set arrive as a pair, and a later
		// --server-set on this step still wins.
		if in.Server.SetPath == "" {
			in.Server.SetPath = ws.State().ServerSet
		}
		resolved, err := resource.ResolveServer(in.Server, minValidators, portBand)
		if err != nil {
			return "", err
		}
		if resolved.HasTarget {
			if err := ws.Retarget(resolved.Target); err != nil {
				return "", err
			}
		}
		return ws.Allocate(AllocateOpts{
			BPCount: in.BPCount, ENCount: in.ENCount, PNCount: in.PNCount,
			EndpointSyncMode: in.EndpointSyncMode, Topology: topo, Blueprint: bp,
			Peering: peeringOf(bp, in.Peering),
			Pool:    resolved.Pool, SetPath: in.Server.SetPath, Binaries: in.Binaries,
			BinaryChains: in.BinaryChains,
			AutoSize:     in.AutoSize,
		})
	})
	return StepOut{Detail: detail}, err
}

// readBlueprint loads a network declaration, or returns nil when none is named.
func readBlueprint(path string) (*blueprint.Blueprint, error) {
	if path == "" {
		return nil, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("chainsetup: read blueprint %s: %w", path, err)
	}
	bp, err := blueprint.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &bp, nil
}

// peeringOf lets a flag override the declaration, and the declaration answer
// when the flag is silent. A flag is a person typing now, which is the one
// thing that outranks a document.
func peeringOf(bp *blueprint.Blueprint, flag string) string {
	if flag != "" || bp == nil {
		return flag
	}
	return bp.Peering
}

// NetGenesis builds the genesis from the key set and writes it to the target.
func NetGenesis(ctx context.Context, d Deps, in NetGenesisIn) (StepOut, error) {
	// The request is read before the workspace is opened: one that contradicts
	// itself needs no workspace, and refusing here keeps the lock and the state
	// out of a request that was never going to be carried out.
	opts, err := genesisOpts(in)
	if err != nil {
		return StepOut{}, err
	}
	return inWorkspace(d, in.DataDir, func(ws *Workspace) (StepOut, error) {
		return ws.Genesis(ctx, opts)
	})
}

// overrides to record before rendering.
type NetConfigIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	// Node scopes a Set override to that 1-based node; 0 means every node.
	Node int `cb:"node" help:"scope --set to this 1-based node (default: every node)"`
	// Set are dot-path "key=value" config-knob overrides to record. Empty
	// renders with whatever overrides the workspace already holds.
	Set []string `cb:"set" help:"config knob override key=value (repeatable; supported keys: syncMode, httpHost, metricsHost)"`
	// ScopedSet records overrides for several scopes at once ("all", a role,
	// "node<N>"), the form the up flow and a DSL env pass. It is applied before
	// Set, most general scope first.
	ScopedSet map[string][]string
}

// NetConfig records any per-node overrides, then renders and writes each node's
// TOML config with them applied. Recording and rendering share one step so
// `chain config --node N --set k=v` both persists the override and reflects it.
func NetConfig(ctx context.Context, d Deps, in NetConfigIn) (StepOut, error) {
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		for _, scope := range sortedScopes(in.ScopedSet) {
			if err := ws.recordConfigSet(scope, in.ScopedSet[scope]); err != nil {
				return "", fmt.Errorf("chainsetup: config: %w", err)
			}
		}
		if len(in.Set) > 0 {
			scope := node.ScopeAll
			if in.Node > 0 {
				scope = fmt.Sprintf("node%d", in.Node)
			}
			if err := ws.recordConfigSet(scope, in.Set); err != nil {
				return "", fmt.Errorf("chainsetup: config: %w", err)
			}
		}
		return ws.Config(ctx)
	})
	return StepOut{Detail: detail}, err
}

// NetLaunchOptsIn customizes the assembled argv.
type NetLaunchOptsIn struct {
	DataDir string   `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	Set     []string `cb:"set" help:"high-precedence launch knob key=value (repeatable; bare key for booleans)"` // key=value overrides for every node (bare key for booleans)
	// ScopedSet records overrides for several scopes at once ("all", a role like
	// "bp"/"en", or "node<N>"), the form the up flow and a DSL env pass. Set is
	// folded into the "all" scope.
	ScopedSet map[string][]string
}

// NetLaunchOptsOut is the assembled per-node argv table.
type NetLaunchOptsOut struct {
	Detail string
	Nodes  []node.Record
}

// NetLaunchOpts assembles each node's launch argv (the single assembly site)
// and records it, returning the table so the surface can render the commands.
func NetLaunchOpts(_ context.Context, d Deps, in NetLaunchOptsIn) (NetLaunchOptsOut, error) {
	var nodes []node.Record
	detail, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		for _, scope := range sortedScopes(in.ScopedSet) {
			if err := ws.recordLaunchSet(scope, in.ScopedSet[scope]); err != nil {
				return "", fmt.Errorf("chainsetup: launchopts: %w", err)
			}
		}
		if err := ws.recordLaunchSet("all", in.Set); err != nil {
			return "", fmt.Errorf("chainsetup: launchopts: %w", err)
		}
		det, err := ws.LaunchOpts()
		nodes = ws.State().Nodes
		return det, err
	})
	return NetLaunchOptsOut{Detail: detail, Nodes: nodes}, err
}

// NetProvisionIn identifies the workspace.
