package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/hardfork"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// Chain-upgrade use cases: plan a binary swap at a fork block over a running
// network, and execute it. Planning is separate from executing so a surface can
// show what will happen before anything is swapped (worklist T7.11).

// HardforkPlanIn describes the upgrade of an already-launched network.
type HardforkPlanIn struct {
	// DataDir is the running network's data root. The from-chain is read from
	// its node set, so it is never guessed.
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
	Plan hardfork.Plan
	// Nodes is the running network the plan was built against.
	Nodes node.NodeSet
	// To is the resolved target chain, so a surface can fall back to its
	// manifest binary name when resolving the executable.
	To registry.ChainPlugin
}

// HardforkPlan resolves the upgrade without touching the network.
func HardforkPlan(_ context.Context, _ Deps, in HardforkPlanIn) (HardforkPlanOut, error) {
	if in.DataDir == "" {
		return HardforkPlanOut{}, errNoDataDir
	}
	if in.ToChain == "" {
		return HardforkPlanOut{}, errors.New("app: hardfork needs a target chain")
	}
	ns, err := session.LoadLocalNodeSet(in.DataDir)
	if err != nil {
		return HardforkPlanOut{}, err
	}
	from, err := registry.Get(ns.Chain)
	if err != nil {
		return HardforkPlanOut{}, fmt.Errorf("app: from-chain %q: %w", ns.Chain, err)
	}
	to, err := registry.Get(in.ToChain)
	if err != nil {
		return HardforkPlanOut{}, err
	}
	if in.ToChain == ns.Chain && in.ToBinary == "" {
		return HardforkPlanOut{}, fmt.Errorf(
			"app: same-chain hardfork (%s -> %s) needs an explicit post-fork binary", ns.Chain, in.ToChain)
	}
	plan, err := hardfork.BuildPlan(ns, from, to, in.Block, in.DataDir)
	if err != nil {
		return HardforkPlanOut{}, err
	}
	return HardforkPlanOut{Plan: plan, Nodes: ns, To: to}, nil
}

// HardforkExecuteIn carries a resolved plan and the binary to swap to.
type HardforkExecuteIn struct {
	Plan HardforkPlanOut
	// DataDir is the network's data root.
	DataDir string
	// Binary is the resolved post-fork executable. As with a launch, this layer
	// does not look it up on PATH.
	Binary string
}

// HardforkExecuteOut is the upgraded network.
type HardforkExecuteOut struct {
	Nodes node.NodeSet
}

// HardforkExecute performs the swap and records the result. It also rewrites
// the saved launch specs to the post-fork binary, so a later single-node
// restart or a further upgrade starts from the binary the network is actually
// running rather than the one it was launched with.
func HardforkExecute(ctx context.Context, d Deps, in HardforkExecuteIn) (HardforkExecuteOut, error) {
	if in.Binary == "" {
		return HardforkExecuteOut{}, errors.New("app: hardfork needs a resolved post-fork binary path")
	}
	specs, err := session.LoadLocalNodeSpecs(in.DataDir)
	if err != nil {
		return HardforkExecuteOut{}, fmt.Errorf("app: load node specs (launch the network first): %w", err)
	}
	dr, err := d.nodeDriver()
	if err != nil {
		return HardforkExecuteOut{}, err
	}
	ns, err := in.Plan.Plan.Execute(ctx, dr, specs, in.Binary)
	if err != nil {
		return HardforkExecuteOut{}, err
	}
	if err := session.SaveLocalNodeSet(in.DataDir, ns); err != nil {
		return HardforkExecuteOut{}, err
	}
	for i := range specs {
		specs[i].Binary = in.Binary
	}
	if err := session.SaveLocalNodeSpecs(in.DataDir, specs); err != nil {
		return HardforkExecuteOut{}, err
	}
	return HardforkExecuteOut{Nodes: ns}, nil
}
