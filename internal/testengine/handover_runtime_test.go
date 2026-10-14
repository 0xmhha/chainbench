package testengine

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestHandoverRefreshesNodesAndWaitsForActualProduction(t *testing.T) {
	before := node.NodeSet{Nodes: []node.Node{{Index: 1, Role: node.RoleEN, PID: 10}}}
	after := node.NodeSet{Nodes: []node.Node{{Index: 1, Role: node.RoleBP, PID: 20}}}
	net := composed{nodes: &before, fork: forkGate{at: 20, preFork: map[int]bool{5: true}}}
	rep := health.Report{Nodes: []health.NodeInfo{{Index: 1, OK: true, BlockNumber: 19}}}
	if !net.fork.parked(rep) {
		t.Fatal("pre-fork parking is required before handover")
	}
	recordHandover(&net, after)
	if net.nodes.Nodes[0].PID != 20 || net.nodes.Nodes[0].Role != node.RoleBP {
		t.Error("stale pre-fork node table", net.nodes.Nodes)
	}
	if net.fork.parked(rep) {
		t.Error("handed-over successors at fork-1 were mistaken for production")
	}
	if !net.fork.retired(5, 20) {
		t.Error("pre-fork producer must remain retired after crossing")
	}
	if before.Nodes[0].PID != 10 {
		t.Error("previous snapshot mutated")
	}
}
