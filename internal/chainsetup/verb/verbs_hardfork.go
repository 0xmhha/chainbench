package verb

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/chainsetup"

	"github.com/0xmhha/chainbench/internal/core/hardfork"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// Chain-upgrade use cases: plan a binary swap at a fork block over a running
// network, and execute it. Planning is separate from executing so a surface can
// show what will happen before anything is swapped.

// HardforkPlanIn describes the upgrade of an already-composed network.
type HardforkPlanIn struct {
	// DataDir is the running network's workspace. The from-chain is read from
	// its record, so it is never guessed.
	DataDir string
	// ToChain is the chain id to upgrade to.
	ToChain string
	// ToBinary is the post-fork node binary. Required for a same-chain
	// upgrade, where both sides resolve to the same manifest binary name and
	// there would otherwise be nothing to swap.
	ToBinary string
	// Block is the fork activation block.
	Block int64
}

// HardforkPlanOut is the resolved swap description.
type HardforkPlanOut struct {
	Plan hardfork.SwapPlan
	// Nodes is the running network the plan was built against.
	Nodes node.NodeSet
	// To is the resolved target chain, so a surface can fall back to its
	// manifest binary name when resolving the executable.
	To registry.ChainPlugin
}

// HardforkPlan resolves the upgrade without touching the network.
func HardforkPlan(_ context.Context, d chainsetup.Deps, in HardforkPlanIn) (HardforkPlanOut, error) {
	if in.DataDir == "" {
		return HardforkPlanOut{}, ErrNoDataDir
	}
	if in.ToChain == "" {
		return HardforkPlanOut{}, errors.New("chainsetup: hardfork needs a target chain")
	}
	ws, err := chainsetup.Open(in.DataDir, d.Clock)
	if err != nil {
		return HardforkPlanOut{}, err
	}
	ns := ws.NodeSet()
	from, err := registry.Get(ns.Chain)
	if err != nil {
		return HardforkPlanOut{}, fmt.Errorf("chainsetup: from-chain %q: %w", ns.Chain, err)
	}
	to, err := registry.Get(in.ToChain)
	if err != nil {
		return HardforkPlanOut{}, err
	}
	if in.ToChain == ns.Chain && in.ToBinary == "" {
		return HardforkPlanOut{}, fmt.Errorf(
			"chainsetup: same-chain hardfork (%s -> %s) needs an explicit post-fork binary", ns.Chain, in.ToChain)
	}
	plan, err := hardfork.PlanSwap(ns, from, to, in.Block, ws.State().Target.DataRoot)
	if err != nil {
		return HardforkPlanOut{}, err
	}
	return HardforkPlanOut{Plan: plan, Nodes: ns, To: to}, nil
}

// HardforkExecuteIn carries a resolved plan and the binary to swap to.
type HardforkExecuteIn struct {
	Plan HardforkPlanOut
	// DataDir is the network's workspace.
	DataDir string
	// Binary is the resolved post-fork executable. As with a launch, this layer
	// does not look it up on PATH.
	Binary string
}

// HardforkExecuteOut is the upgraded network.
type HardforkExecuteOut struct {
	Nodes node.NodeSet
}

// HardforkExecute performs the swap and records the result on the workspace:
// each node's new pid, the binary the network now runs, and the chain it now
// is — so a later single-node restart or a further upgrade starts from what
// is actually running rather than what was launched.
//
// The relaunch reuses each node's armed argv. That is load-bearing: the argv
// carries the node's validator identity and its peering, and a node relaunched
// on generic flags would rejoin consensus as an unauthorized address.
func HardforkExecute(ctx context.Context, d chainsetup.Deps, in HardforkExecuteIn) (HardforkExecuteOut, error) {
	if in.Binary == "" {
		return HardforkExecuteOut{}, errors.New("chainsetup: hardfork needs a resolved post-fork binary path")
	}
	ws, err := chainsetup.Open(in.DataDir, d.Clock)
	if err != nil {
		return HardforkExecuteOut{}, err
	}
	ns, err := chainsetup.NewManager(d, ws).Hardfork(ctx, in.Plan.Plan, in.Binary)
	if err != nil {
		return HardforkExecuteOut{}, err
	}
	return HardforkExecuteOut{Nodes: ns}, nil
}

// Hardfork swaps every node onto binary at the plan's fork, continuing the
