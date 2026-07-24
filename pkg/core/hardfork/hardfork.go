// Package hardfork plans a chain upgrade that reproduces ../script/wemix-upgrade:
// the same node data directories are re-run with a different binary that
// activates a fork at a given block (e.g. wemix/gwemix -> wbft/geth at the
// montBlanc block). It builds the swap as inspectable data; executing it
// (stop + relaunch on the new binary) needs PID tracking and built binaries and
// is a later slice (docs/CHAINBENCH_GO_REDESIGN.md §3.1, §11).
package hardfork

import (
	"fmt"
	"path/filepath"

	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

// NodeSwap is one node's binary swap, keeping its data directory.
type NodeSwap struct {
	Index      int
	DataDir    string
	FromBinary string
	ToBinary   string
}

// Plan is a full hardfork upgrade description.
type Plan struct {
	FromChain  string
	ToChain    string
	FromBinary string
	ToBinary   string
	Block      int64
	Swaps      []NodeSwap
}

// BuildPlan builds a hardfork upgrade from a running from-chain NodeSet to a
// target chain, activating at block. dataRoot locates each node's datadir
// (dataRoot/node<index>), which is preserved across the swap.
func BuildPlan(ns node.NodeSet, from, to registry.ChainPlugin, block int64, dataRoot string) (Plan, error) {
	if len(ns.Nodes) == 0 {
		return Plan{}, fmt.Errorf("hardfork: empty node set")
	}
	if block < 0 {
		return Plan{}, fmt.Errorf("hardfork: negative block %d", block)
	}
	fromBin := from.Manifest().Binary
	toBin := to.Manifest().Binary
	if fromBin == toBin {
		return Plan{}, fmt.Errorf("hardfork: from and to use the same binary %q; nothing to swap", fromBin)
	}

	swaps := make([]NodeSwap, 0, len(ns.Nodes))
	for _, n := range ns.Nodes {
		swaps = append(swaps, NodeSwap{
			Index:      n.Index,
			DataDir:    filepath.Join(dataRoot, fmt.Sprintf("node%d", n.Index)),
			FromBinary: fromBin,
			ToBinary:   toBin,
		})
	}
	return Plan{
		FromChain:  from.Manifest().ID,
		ToChain:    to.Manifest().ID,
		FromBinary: fromBin,
		ToBinary:   toBin,
		Block:      block,
		Swaps:      swaps,
	}, nil
}
