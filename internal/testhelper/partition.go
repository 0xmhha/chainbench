package testhelper

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/dsl/interp"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// Splitting a network in two, and putting it back.
//
// A partition is expressed as groups of nodes rather than as a list of links,
// because what a test means is "these cannot see those" and the peer set is how
// that is said. Healing re-adds the peers the split removed, so a case can
// watch a chain recover rather than only fail.

type partitionAction struct{}

func (partitionAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
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
						return fmt.Errorf("dsl: partition: node%d drop node%d: %w", from.Index, to.Index, err)
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

func (healPartitionAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	if ac.Env == nil {
		return fmt.Errorf("dsl: healPartition: no environment")
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
		return fmt.Errorf("dsl: healPartition needs at least 2 nodes, got %d", len(nodes))
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
				return fmt.Errorf("dsl: healPartition: node%d add node%d: %w", from.Index, to.Index, err)
			}
		}
	}
	return nil
}

// partitionGroups resolves and validates the action's "groups" argument.
func partitionGroups(ac *interp.ActionCtx) ([][]node.Node, error) {
	raw, ok := ac.Args["groups"]
	if !ok {
		return nil, fmt.Errorf("dsl: partition requires \"groups\" (two or more lists of node selectors)")
	}
	groups, err := resolveGroups(ac, raw)
	if err != nil {
		return nil, err
	}
	if len(groups) < 2 {
		return nil, fmt.Errorf("dsl: partition needs at least 2 groups, got %d", len(groups))
	}
	for i, g := range groups {
		if len(g) == 0 {
			return nil, fmt.Errorf("dsl: partition: group %d is empty", i)
		}
	}
	return groups, nil
}

// resolveGroups turns the DSL's [[selector...]...] into resolved node groups.
func resolveGroups(ac *interp.ActionCtx, raw any) ([][]node.Node, error) {
	if ac.Env == nil {
		return nil, fmt.Errorf("dsl: partition: no environment")
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("dsl: partition: \"groups\" must be a list of node-selector lists")
	}
	out := make([][]node.Node, 0, len(list))
	for gi, g := range list {
		sels, ok := g.([]any)
		if !ok {
			return nil, fmt.Errorf("dsl: partition: group %d must be a list of node selectors", gi)
		}
		nodes := make([]node.Node, 0, len(sels))
		for _, s := range sels {
			sel, ok := s.(string)
			if !ok {
				return nil, fmt.Errorf("dsl: partition: group %d has a non-string selector", gi)
			}
			n, err := ac.Env.Resolve(sel)
			if err != nil {
				return nil, fmt.Errorf("dsl: partition: %w", err)
			}
			nodes = append(nodes, n)
		}
		out = append(out, nodes)
	}
	return out, nil
}

// enodesFor asks each node for its own enode (admin_nodeInfo), keyed by index.
// Peers are named by enode, and only the node itself knows its own.
func enodesFor(ctx context.Context, ac *interp.ActionCtx, nodes []node.Node) (map[int]string, error) {
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
			return nil, fmt.Errorf("dsl: enode of node%d: %w", n.Index, err)
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

// rpc.Client must satisfy what the actions need; this fails to compile if
// it stops.
var _ = func(c *rpc.Client) { _ = c.AddPeer }
