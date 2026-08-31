package chainsetup

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/genesis"
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
	Plan genesis.Hardfork
	// Nodes is the running network the plan was built against.
	Nodes node.NodeSet
	// To is the resolved target chain, so a surface can fall back to its
	// manifest binary name when resolving the executable.
	To registry.ChainPlugin
}

// HardforkPlan resolves the upgrade without touching the network.
func HardforkPlan(_ context.Context, d Deps, in HardforkPlanIn) (HardforkPlanOut, error) {
	if in.DataDir == "" {
		return HardforkPlanOut{}, ErrNoDataDir
	}
	if in.ToChain == "" {
		return HardforkPlanOut{}, errors.New("chainsetup: hardfork needs a target chain")
	}
	ws, err := Open(in.DataDir, d.Clock)
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
	plan, err := genesis.PlanHardfork(ns, from, to, in.Block, ws.State().Target.DataRoot)
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
func HardforkExecute(ctx context.Context, d Deps, in HardforkExecuteIn) (HardforkExecuteOut, error) {
	if in.Binary == "" {
		return HardforkExecuteOut{}, errors.New("chainsetup: hardfork needs a resolved post-fork binary path")
	}
	var out HardforkExecuteOut
	_, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		ns, err := ws.Hardfork(ctx, in.Plan.Plan, in.Binary)
		if err != nil {
			return "", err
		}
		out.Nodes = ns
		return fmt.Sprintf("%d node(s) now on %s (%s)", len(ns.Nodes), in.Plan.Plan.ToChain, in.Binary), nil
	})
	return out, err
}

// Hardfork swaps every node onto binary at the plan's fork, continuing the
// same chain data, and records the new pids, binary and chain.
func (w *Workspace) Hardfork(ctx context.Context, plan genesis.Hardfork, binary string) (node.NodeSet, error) {
	if len(w.state.Nodes) == 0 {
		return node.NodeSet{}, fmt.Errorf("chainsetup: hardfork: no node table — compose the network first")
	}
	specs := make([]driver.NodeSpec, 0, len(w.state.Nodes))
	for _, rec := range w.state.Nodes {
		if len(rec.Args) == 0 {
			return node.NodeSet{}, fmt.Errorf("chainsetup: hardfork: node%d has no recorded argv — run `net start` first", rec.Index)
		}
		spec := driver.SpecOf(rec)
		spec.Binary = w.state.Binary
		specs = append(specs, spec)
	}
	// One driver relaunches every node: the swap runs on the machine the
	// network's nodes share. (A network spread across a server set would need
	// a per-node driver; the plan executes over one.)
	t, err := w.machineFor(w.state.Nodes[0])
	if err != nil {
		return node.NodeSet{}, err
	}
	ns, err := plan.Execute(ctx, t.Driver, specs, binary)
	if err != nil {
		return ns, err
	}
	for _, n := range ns.Nodes {
		for i, rec := range w.state.Nodes {
			if rec.Index != n.Index {
				continue
			}
			w.clearPID(i)
			if err := w.recordLaunch(i, n.PID, binary); err != nil {
				return ns, fmt.Errorf("chainsetup: hardfork: node%d: %w", n.Index, err)
			}
		}
	}
	w.state.Chain = plan.ToChain
	w.state.Binary = binary
	w.markStep("hardfork", fmt.Sprintf("%s -> %s at block %d on %s", plan.FromChain, plan.ToChain, plan.Block, binary))
	return ns, nil
}
