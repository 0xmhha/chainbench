package testspec

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// Fault and node-lifecycle action names (design §3.2). These are what a
// destructive test uses: stop a validator to probe quorum, restart it to check
// sync recovery, or split the network to induce a fork.
const (
	actionStopNode      = "stopNode"
	actionStartNode     = "startNode"
	actionRestartNode   = "restartNode"
	actionPartition     = "partition"
	actionHealPartition = "healPartition"
)

// NodeControl stops and restarts individual node processes. It is the seam
// between the DSL and process management: the local engine wires an
// implementation backed by the launcher and procman, while attach mode leaves it
// nil because chainbench did not start those nodes and must not pretend it can
// stop them.
type NodeControl interface {
	// Stop terminates the node's process, verifying it is gone.
	Stop(ctx context.Context, n node.Node) error
	// Start relaunches a previously stopped node with its original arming.
	Start(ctx context.Context, n node.Node) error
}

// seedFaultBuiltins registers the node-lifecycle and partition actions.
func seedFaultBuiltins(r Registry) {
	r.RegisterAction(actionStopNode, stopNodeAction{})
	r.RegisterAction(actionStartNode, startNodeAction{})
	r.RegisterAction(actionRestartNode, restartNodeAction{})
	r.RegisterAction(actionPartition, partitionAction{})
	r.RegisterAction(actionHealPartition, healPartitionAction{})
}

// stopNodeAction stops one node. Args: on (selector, required).
type stopNodeAction struct{}

func (stopNodeAction) Do(ctx context.Context, ac *ActionCtx) error {
	n, ctrl, err := faultTarget(ac, actionStopNode)
	if err != nil {
		return err
	}
	if err := ctrl.Stop(ctx, n); err != nil {
		return fmt.Errorf("testspec: stopNode node%d: %w", n.Index, err)
	}
	return nil
}

// startNodeAction starts one previously stopped node. Args: on (required).
type startNodeAction struct{}

func (startNodeAction) Do(ctx context.Context, ac *ActionCtx) error {
	n, ctrl, err := faultTarget(ac, actionStartNode)
	if err != nil {
		return err
	}
	if err := ctrl.Start(ctx, n); err != nil {
		return fmt.Errorf("testspec: startNode node%d: %w", n.Index, err)
	}
	return nil
}

// restartNodeAction stops then starts one node — the sync-recovery scenario.
// Args: on (required).
type restartNodeAction struct{}

func (restartNodeAction) Do(ctx context.Context, ac *ActionCtx) error {
	n, ctrl, err := faultTarget(ac, actionRestartNode)
	if err != nil {
		return err
	}
	if err := ctrl.Stop(ctx, n); err != nil {
		return fmt.Errorf("testspec: restartNode node%d: stop: %w", n.Index, err)
	}
	if err := ctrl.Start(ctx, n); err != nil {
		return fmt.Errorf("testspec: restartNode node%d: start: %w", n.Index, err)
	}
	return nil
}

// faultTarget resolves the action's "on" selector and the injected node control,
// naming whichever is missing.
func faultTarget(ac *ActionCtx, action string) (node.Node, NodeControl, error) {
	if ac.Deps == nil || ac.Deps.Nodes == nil {
		return node.Node{}, nil, fmt.Errorf(
			"testspec: %s needs node control, which this run has none of (attach mode does not own the node processes)", action)
	}
	if ac.Env == nil {
		return node.Node{}, nil, fmt.Errorf("testspec: %s: no environment", action)
	}
	sel, _ := ac.Args["on"].(string)
	if sel == "" {
		return node.Node{}, nil, fmt.Errorf("testspec: %s requires an \"on\" selector", action)
	}
	n, err := ac.Env.Resolve(sel)
	if err != nil {
		return node.Node{}, nil, fmt.Errorf("testspec: %s: %w", action, err)
	}
	return n, ac.Deps.Nodes, nil
}

// partitionAction splits the network by dropping every peer link that crosses a
// group boundary, so the groups can only see themselves. This is how a spec
// induces a fork to check the cross-node divergence assertions (F8 AC-2).
//
// Args: groups — two or more lists of node selectors, e.g.
//
//	{"partition": {"groups": [["bp1","bp2"], ["bp3","bp4"]]}}
//
// Links are severed from BOTH sides, because admin_removePeer only drops the
// connection the node it is called on holds. Nodes not named in any group are
// left alone.
//
// Note: this severs current connections. A network with peer discovery enabled
// may re-establish them; chainbench networks peer through static-nodes, so the
// split holds until healPartition (or a node restart) restores it.
type partitionAction struct{}

func (partitionAction) Do(ctx context.Context, ac *ActionCtx) error {
	groups, err := partitionGroups(ac)
	if err != nil {
		return err
	}
	enodes, err := enodesFor(ctx, ac, flatten(groups))
	if err != nil {
		return err
	}
	for i, a := range groups {
		for j, b := range groups {
			if i == j {
				continue
			}
			for _, from := range a {
				for _, to := range b {
					c, err := clientFor(ac.Deps, from.RPCURL)
					if err != nil {
						return err
					}
					if err := c.RemovePeer(ctx, enodes[to.Index]); err != nil {
						return fmt.Errorf("testspec: partition: node%d drop node%d: %w", from.Index, to.Index, err)
					}
				}
			}
		}
	}
	return nil
}

// healPartitionAction restores full connectivity by re-adding every pair. With
// no "groups" it heals across the whole environment, which is what a post-action
// wants after a fault test.
type healPartitionAction struct{}

func (healPartitionAction) Do(ctx context.Context, ac *ActionCtx) error {
	if ac.Env == nil {
		return fmt.Errorf("testspec: healPartition: no environment")
	}
	nodes := ac.Env.Nodes()
	if raw, ok := ac.Args["groups"]; ok {
		groups, err := resolveGroups(ac, raw)
		if err != nil {
			return err
		}
		nodes = flatten(groups)
	}
	if len(nodes) < 2 {
		return fmt.Errorf("testspec: healPartition needs at least 2 nodes, got %d", len(nodes))
	}
	enodes, err := enodesFor(ctx, ac, nodes)
	if err != nil {
		return err
	}
	for _, from := range nodes {
		c, err := clientFor(ac.Deps, from.RPCURL)
		if err != nil {
			return err
		}
		for _, to := range nodes {
			if from.Index == to.Index {
				continue
			}
			if err := c.AddPeer(ctx, enodes[to.Index]); err != nil {
				return fmt.Errorf("testspec: healPartition: node%d add node%d: %w", from.Index, to.Index, err)
			}
		}
	}
	return nil
}

// partitionGroups resolves and validates the action's "groups" argument.
func partitionGroups(ac *ActionCtx) ([][]node.Node, error) {
	raw, ok := ac.Args["groups"]
	if !ok {
		return nil, fmt.Errorf("testspec: partition requires \"groups\" (two or more lists of node selectors)")
	}
	groups, err := resolveGroups(ac, raw)
	if err != nil {
		return nil, err
	}
	if len(groups) < 2 {
		return nil, fmt.Errorf("testspec: partition needs at least 2 groups, got %d", len(groups))
	}
	for i, g := range groups {
		if len(g) == 0 {
			return nil, fmt.Errorf("testspec: partition: group %d is empty", i)
		}
	}
	return groups, nil
}

// resolveGroups turns the DSL's [[selector...]...] into resolved node groups.
func resolveGroups(ac *ActionCtx, raw any) ([][]node.Node, error) {
	if ac.Env == nil {
		return nil, fmt.Errorf("testspec: partition: no environment")
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("testspec: partition: \"groups\" must be a list of node-selector lists")
	}
	out := make([][]node.Node, 0, len(list))
	for gi, g := range list {
		sels, ok := g.([]any)
		if !ok {
			return nil, fmt.Errorf("testspec: partition: group %d must be a list of node selectors", gi)
		}
		nodes := make([]node.Node, 0, len(sels))
		for _, s := range sels {
			sel, ok := s.(string)
			if !ok {
				return nil, fmt.Errorf("testspec: partition: group %d has a non-string selector", gi)
			}
			n, err := ac.Env.Resolve(sel)
			if err != nil {
				return nil, fmt.Errorf("testspec: partition: %w", err)
			}
			nodes = append(nodes, n)
		}
		out = append(out, nodes)
	}
	return out, nil
}

// enodesFor asks each node for its own enode (admin_nodeInfo), keyed by index.
// Peers are named by enode, and only the node itself knows its own.
func enodesFor(ctx context.Context, ac *ActionCtx, nodes []node.Node) (map[int]string, error) {
	out := make(map[int]string, len(nodes))
	for _, n := range nodes {
		if _, done := out[n.Index]; done {
			continue
		}
		c, err := clientFor(ac.Deps, n.RPCURL)
		if err != nil {
			return nil, err
		}
		enode, err := c.Enode(ctx)
		if err != nil {
			return nil, fmt.Errorf("testspec: enode of node%d: %w", n.Index, err)
		}
		out[n.Index] = enode
	}
	return out, nil
}

// flatten concatenates node groups, preserving order.
func flatten(groups [][]node.Node) []node.Node {
	var out []node.Node
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// compile-time assertion that the RPC client satisfies what the actions need.
var _ = func(c *rpc.Client) { _ = c.AddPeer }
