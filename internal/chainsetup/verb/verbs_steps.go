package verb

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// The composition verbs, as a surface calls them: keys, allocate, genesis,
// config, launch options.
//
// These carry more than a call because a declaration reaches them here — a
// blueprint to read, an overlay to stage, a peering to resolve. The lifecycle

// ChainKeysIn selects where node identities come from.
type ChainKeysIn struct {
	DataDir string `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	// Source is preset (default) | generate | declared.
	Source string `cb:"keys-source" default:"keyPreset" help:"keyPreset (use the recorded key set) | generate (create a fresh set)"`
	// BlueprintPath is the declaration the keys come from when Source is
	// "declared" (or when a blueprint is given and Source is silent).
	BlueprintPath string
	Nodes         int `cb:"nodes" help:"identities the set must cover (default: the allocated node count)"`
	Validators    int `cb:"validators" help:"identities joining the validator set (generate; 0 = all)"`
}

// ChainKeys ensures the workspace's key set exists and covers the node count.
func ChainKeys(ctx context.Context, d chainsetup.Deps, in ChainKeysIn) (chainsetup.StepOut, error) {
	return step(ctx, d, in.DataDir, "keys", chainsetup.ChainUpIn{
		KeysSource: in.Source, BlueprintPath: in.BlueprintPath,
		KeysNodes: in.Nodes, KeysValidators: in.Validators,
	})
}

// ChainAllocate builds the node table (roles, paths, deterministic ports).
func ChainAllocate(ctx context.Context, d chainsetup.Deps, in chainsetup.ChainAllocateIn) (chainsetup.StepOut, error) {
	return step(ctx, d, in.DataDir, "place", chainsetup.ChainUpIn{
		BPCount: in.BPCount, ENCount: in.ENCount, PNCount: in.PNCount,
		EndpointSyncMode: in.EndpointSyncMode, TopologyPath: in.TopologyPath,
		BlueprintPath: in.BlueprintPath, Topology: in.Topology, Peering: in.Peering,
		Binaries: in.Binaries, BinaryChains: in.BinaryChains,
		Server: in.Server, AutoSize: in.AutoSize,
	})
}

// ChainGenesis builds the genesis from the key set and writes it to the target.
func ChainGenesis(ctx context.Context, d chainsetup.Deps, in chainsetup.ChainGenesisIn) (chainsetup.StepOut, error) {
	// Read before the workspace is opened. A request that contradicts itself
	// needs no workspace, and refusing here is what lets `chain genesis
	// --existing X --chain-id 7` be refused without one. The stage reads it
	// again; it is a pure check of the request and gives the same answer.
	if _, err := chainsetup.GenesisOptsFor(in); err != nil {
		return chainsetup.StepOut{}, err
	}
	return step(ctx, d, in.DataDir, "genesis", chainsetup.ChainUpIn{
		ChainID: in.ChainID, GenesisSet: in.Set, OverlayPath: in.OverlayPath,
		GenesisExisting: in.GenesisExisting, GenesisPerBinary: in.PerBinary,
		GenesisFork: in.Fork,
	})
}

// overrides to record before rendering.
type ChainConfigIn struct {
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

// ChainConfig records any per-node overrides, then renders and writes each node's
// TOML config with them applied. Recording and rendering share one step so
// `chain config --node N --set k=v` both persists the override and reflects it.
func ChainConfig(ctx context.Context, d chainsetup.Deps, in ChainConfigIn) (chainsetup.StepOut, error) {
	set := map[string][]string{}
	for scope, values := range in.ScopedSet {
		set[scope] = values
	}
	if len(in.Set) > 0 {
		// --node N --set k=v is the node's scope written the short way.
		scope := node.ScopeAll
		if in.Node > 0 {
			scope = fmt.Sprintf("node%d", in.Node)
		}
		set[scope] = append(set[scope], in.Set...)
	}
	return step(ctx, d, in.DataDir, "config", chainsetup.ChainUpIn{ConfigSet: set})
}

// ChainLaunchOptsIn customizes the assembled argv.
type ChainLaunchOptsIn struct {
	DataDir string   `cb:"workspace-dir,required" help:"workspace directory (where the composition is set up)"`
	Set     []string `cb:"set" help:"high-precedence launch knob key=value (repeatable; bare key for booleans)"` // key=value overrides for every node (bare key for booleans)
	// ScopedSet records overrides for several scopes at once ("all", a role like
	// "bp"/"en", or "node<N>"), the form the up flow and a DSL env pass. Set is
	// folded into the "all" scope.
	ScopedSet map[string][]string
}

// ChainLaunchOptsOut is the assembled per-node argv table.
type ChainLaunchOptsOut struct {
	Detail string
	Nodes  []node.Record
}

// ChainLaunchOpts assembles each node's launch argv (the single assembly site)
// and records it, returning the table so the surface can render the commands.
func ChainLaunchOpts(ctx context.Context, d chainsetup.Deps, in ChainLaunchOptsIn) (ChainLaunchOptsOut, error) {
	out, err := step(ctx, d, in.DataDir, "build", chainsetup.ChainUpIn{
		LaunchSet: in.Set, LaunchScoped: in.ScopedSet,
	})
	if err != nil {
		return ChainLaunchOptsOut{}, err
	}
	// The table is read back rather than carried out of the step: what the
	// surface renders is the node records as they now stand.
	nodes, nerr := chainsetup.InWorkspace(d, in.DataDir, func(ws *chainsetup.Workspace) ([]node.Record, error) {
		return ws.State().Nodes, nil
	})
	return ChainLaunchOptsOut{Detail: out.Detail, Nodes: nodes}, nerr
}

// ChainProvisionIn identifies the workspace.
