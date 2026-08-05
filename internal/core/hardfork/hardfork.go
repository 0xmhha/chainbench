// Package hardfork plans a binary-SWAP upgrade: the same node data directories
// are stopped and re-run with a different binary that activates a fork at a
// given block. This fits a homogeneous fork where every node upgrades in place
// and the consensus engine is unchanged across the fork.
//
// It does NOT fit a consensus-family handoff such as go-wemix+etcd (poa) ->
// go-wbft (bft), where the two binaries must run CONCURRENTLY with disjoint
// roles (producers mine up to the fork; separately-launched validators sync the
// pre-fork chain and take over after it). That verified model lives in
// pkg/consensus/upgrade (BuildPlan/Launch); use it for engine-changing handoffs.
package hardfork

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// NodeSwap is one node's binary swap, keeping its data directory. It carries
// enough to stop the running node (PID) and relaunch it on the new binary
// (datadir/config/ports/role).
type NodeSwap struct {
	Index      int
	Role       node.Role
	DataDir    string
	ConfigPath string
	LogPath    string
	Ports      node.Endpoints
	PID        int
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
	// Note: from and to may share a manifest binary name — a same-chain version
	// swap (e.g. a pre-fork gstable -> a post-fork gstable) is a valid hardfork.
	// What must differ is the actual binary the nodes are relaunched on; the CLI
	// enforces that (a same-chain swap requires an explicit --to-binary path).

	swaps := make([]NodeSwap, 0, len(ns.Nodes))
	for _, n := range ns.Nodes {
		swaps = append(swaps, NodeSwap{
			Index:      n.Index,
			Role:       n.Role,
			DataDir:    filepath.Join(dataRoot, fmt.Sprintf("node%d", n.Index)),
			ConfigPath: filepath.Join(dataRoot, fmt.Sprintf("config_node%d.toml", n.Index)),
			LogPath:    filepath.Join(dataRoot, "logs", fmt.Sprintf("node%d.log", n.Index)),
			Ports:      n.Ports,
			PID:        n.PID,
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

// Execute performs the upgrade: it stops each running node (by PID) and
// relaunches it on binary (the resolved to-chain binary) over the same data
// directory, reusing the node's ORIGINAL launch spec (from setup's
// nodespecs.json) and swapping only the binary. It continues the same chain
// data — no re-init — so the fork activates at the plan's block. It returns the
// new NodeSet (with new PIDs and the to-chain id).
//
// Reusing the original spec is load-bearing: those args carry the node's
// validator identity (--nodekey, --unlock, keystore dir) and its static-nodes
// peering. A homogeneous fork keeps that identity; regenerating generic start
// flags would drop it and the relaunched node would rejoin WBFT consensus as an
// unauthorized address, halting block production.
func (p Plan) Execute(ctx context.Context, d driver.Driver, specs []driver.NodeSpec, binary string) (node.NodeSet, error) {
	byIndex := make(map[int]driver.NodeSpec, len(specs))
	for _, s := range specs {
		byIndex[s.Index] = s
	}
	ns := node.NodeSet{Chain: p.ToChain, Network: "local"}
	for _, s := range p.Swaps {
		if s.PID > 0 {
			// Best-effort stop; a node that already exited is fine.
			_ = d.Stop(ctx, driver.Handle{Index: s.Index, PID: s.PID})
		}
		orig, ok := byIndex[s.Index]
		if !ok {
			return ns, fmt.Errorf("hardfork: no saved spec for node%d; run setup so nodespecs.json exists", s.Index)
		}
		orig.Binary = binary // swap the binary; keep identity/config/peering args
		h, err := d.Launch(ctx, orig)
		if err != nil {
			return ns, fmt.Errorf("hardfork: relaunch node%d on %s: %w", s.Index, binary, err)
		}
		ns.Nodes = append(ns.Nodes, node.Node{
			Index:  s.Index,
			Role:   s.Role,
			Host:   "127.0.0.1",
			RPCURL: fmt.Sprintf("http://127.0.0.1:%d", s.Ports.HTTP),
			Ports:  s.Ports,
			PID:    h.PID,
		})
	}
	return ns, nil
}
