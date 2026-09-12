package testengine

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestFactsFromReport_FillsTheWantsTheClassifierChecks is the test the gate never
// had. The classifier has conditions for peering, for being behind the network,
// for the wrong chain id and for consensus participation; this mapping set only
// Wanted, so three of them could not fire from the run path no matter what the
// network did. A condition that is written, unit-tested in the classifier and
// unreachable from the caller is the same defect as a comparison whose want side
// is never supplied.
func TestFactsFromReport_FillsTheWantsTheClassifierChecks(t *testing.T) {
	ns := node.NodeSet{Nodes: []node.Node{
		{Index: 1, PID: 101}, {Index: 2, PID: 102}, {Index: 3, PID: 103}, {Index: 4, PID: 104},
	}}
	rep := health.Report{
		Producing: true,
		Nodes: []health.NodeInfo{
			{Index: 1, OK: true, BlockNumber: 30, PeerCount: 3},
			{Index: 2, OK: true, BlockNumber: 30, PeerCount: 3},
			{Index: 3, OK: true, BlockNumber: 29, PeerCount: 3},
			{Index: 4, OK: true, BlockNumber: 0, PeerCount: 0},
		},
	}
	facts := factsFromReport(rep, ns)
	if len(facts) != 4 {
		t.Fatalf("got %d facts, want 4", len(facts))
	}
	byNode := map[int]int{}
	for i, f := range facts {
		byNode[f.Node] = i
	}
	for _, f := range facts {
		if f.WantHeight != 30 {
			t.Errorf("node%d: WantHeight = %d, want 30 (the network's head this round)", f.Node, f.WantHeight)
		}
		if f.WantPeers != 1 {
			t.Errorf("node%d: WantPeers = %d, want 1 for a multi-node network", f.Node, f.WantPeers)
		}
	}
	// The node that is alone and at zero must be distinguishable from the three
	// that are not; that is the whole point of filling the wants.
	lone := facts[byNode[4]]
	if lone.Peers != 0 || lone.Height != 0 {
		t.Fatalf("node4 facts did not carry its own observation: %+v", lone)
	}
}

// TestFactsFromReport_ASingleNodeNetworkWantsNoPeer keeps the floor from refusing
// a network that is correct: one node has no one to peer with.
func TestFactsFromReport_ASingleNodeNetworkWantsNoPeer(t *testing.T) {
	facts := factsFromReport(
		health.Report{Producing: true, Nodes: []health.NodeInfo{{Index: 1, OK: true, BlockNumber: 5}}},
		node.NodeSet{Nodes: []node.Node{{Index: 1, PID: 1}}},
	)
	if len(facts) != 1 {
		t.Fatalf("got %d facts, want 1", len(facts))
	}
	if facts[0].WantPeers != 0 {
		t.Errorf("WantPeers = %d, want 0 for a one-node network", facts[0].WantPeers)
	}
}

// TestFactsFromReport_TheHeadIsTheMaximumNotTheFirst pins which node's height
// becomes the network's head. Taking the first would let a lagging node define
// the target and hide exactly the case this was written for.
func TestFactsFromReport_TheHeadIsTheMaximumNotTheFirst(t *testing.T) {
	facts := factsFromReport(
		health.Report{Producing: true, Nodes: []health.NodeInfo{
			{Index: 1, OK: true, BlockNumber: 0},
			{Index: 2, OK: true, BlockNumber: 41},
		}},
		node.NodeSet{Nodes: []node.Node{{Index: 1, PID: 1}, {Index: 2, PID: 2}}},
	)
	for _, f := range facts {
		if f.WantHeight != 41 {
			t.Errorf("node%d: WantHeight = %d, want 41", f.Node, f.WantHeight)
		}
	}
}
