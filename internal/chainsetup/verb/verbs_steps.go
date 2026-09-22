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
	opts, err := chainsetup.KeysOptsFor(in.BlueprintPath, in.Source, in.Nodes, in.Validators)
	if err != nil {
		return chainsetup.StepOut{}, err
	}
	return chainsetup.InWorkspace(d, in.DataDir, func(ws *chainsetup.Workspace) (chainsetup.StepOut, error) {
		return ws.Keys(ctx, opts)
	})
}

// ChainAllocate builds the node table (roles, paths, deterministic ports).
func ChainAllocate(_ context.Context, d chainsetup.Deps, in chainsetup.ChainAllocateIn) (chainsetup.StepOut, error) {
	detail, err := chainsetup.PlaceNodes(d, in)
	return chainsetup.StepOut{Detail: detail}, err
}

// ChainGenesis builds the genesis from the key set and writes it to the target.
func ChainGenesis(ctx context.Context, d chainsetup.Deps, in chainsetup.ChainGenesisIn) (chainsetup.StepOut, error) {
	// The request is read before the workspace is opened: one that contradicts
	// itself needs no workspace, and refusing here keeps the lock and the state
	// out of a request that was never going to be carried out.
	opts, err := chainsetup.GenesisOptsFor(in)
	if err != nil {
		return chainsetup.StepOut{}, err
	}
	return chainsetup.InWorkspace(d, in.DataDir, func(ws *chainsetup.Workspace) (chainsetup.StepOut, error) {
		return ws.Genesis(ctx, opts)
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
	detail, err := chainsetup.WithWorkspace(d, in.DataDir, func(ws *chainsetup.Workspace) (string, error) {
		for _, scope := range sortedScopes(in.ScopedSet) {
			if err := ws.RecordConfigSet(scope, in.ScopedSet[scope]); err != nil {
				return "", fmt.Errorf("chainsetup: config: %w", err)
			}
		}
		if len(in.Set) > 0 {
			scope := node.ScopeAll
			if in.Node > 0 {
				scope = fmt.Sprintf("node%d", in.Node)
			}
			if err := ws.RecordConfigSet(scope, in.Set); err != nil {
				return "", fmt.Errorf("chainsetup: config: %w", err)
			}
		}
		return ws.Config(ctx)
	})
	return chainsetup.StepOut{Detail: detail}, err
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
func ChainLaunchOpts(_ context.Context, d chainsetup.Deps, in ChainLaunchOptsIn) (ChainLaunchOptsOut, error) {
	var nodes []node.Record
	detail, err := chainsetup.WithWorkspace(d, in.DataDir, func(ws *chainsetup.Workspace) (string, error) {
		for _, scope := range sortedScopes(in.ScopedSet) {
			if err := ws.RecordLaunchSet(scope, in.ScopedSet[scope]); err != nil {
				return "", fmt.Errorf("chainsetup: launchopts: %w", err)
			}
		}
		if err := ws.RecordLaunchCommand(in.Set); err != nil {
			return "", fmt.Errorf("chainsetup: launchopts: %w", err)
		}
		det, err := ws.LaunchOpts()
		nodes = ws.State().Nodes
		return det, err
	})
	return ChainLaunchOptsOut{Detail: detail, Nodes: nodes}, err
}

// ChainProvisionIn identifies the workspace.
