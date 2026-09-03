package testengine

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/health"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/nodemonitor"
)

// TestFactsFromReport_MapsAndClassifies: a producing report with a live pid maps
// to a READY node, while a report node whose pid is not recorded maps to a
// process-not-alive fact that classifies RESTARTABLE.
func TestFactsFromReport_MapsAndClassifies(t *testing.T) {
	ns := node.NodeSet{Nodes: []node.Node{
		{Index: 1, PID: 4242},
		{Index: 2, PID: 0}, // stopped: no recorded pid
	}}
	rep := health.Report{
		Producing: true,
		Nodes: []health.NodeInfo{
			{Index: 1, ChainID: 1000, BlockNumber: 12, PeerCount: 3, Syncing: false, OK: true},
			{Index: 2, ChainID: 1000, BlockNumber: 0, PeerCount: 0, Syncing: false, OK: false},
		},
	}
	facts := factsFromReport(rep, ns)
	if len(facts) != 2 {
		t.Fatalf("got %d facts, want 2", len(facts))
	}

	// node1: producing, rpc up, pid alive -> READY.
	if facts[0].Node != 1 || !facts[0].PIDAlive || !facts[0].RPCUp || !facts[0].Advancing {
		t.Fatalf("node1 facts = %+v", facts[0])
	}
	if v := nodemonitor.Classify(facts[0]).Verdict; v != nodemonitor.Ready {
		t.Errorf("node1 verdict = %s, want READY", v)
	}
	if facts[0].ChainID != 1000 || facts[0].Height != 12 || facts[0].Peers != 3 {
		t.Errorf("node1 mapping = chain %d height %d peers %d", facts[0].ChainID, facts[0].Height, facts[0].Peers)
	}

	// node2: no pid, rpc down -> RESTARTABLE.
	if facts[1].PIDAlive || facts[1].RPCUp {
		t.Fatalf("node2 should be dead/rpc-down, got %+v", facts[1])
	}
	if v := nodemonitor.Classify(facts[1]).Verdict; v != nodemonitor.Restartable {
		t.Errorf("node2 verdict = %s, want RESTARTABLE", v)
	}
}

// TestFactsFromReport_NotProducingIsWaitable: rpc up but the network is not yet
// producing maps to WAITABLE, so the gate waits rather than restarts.
func TestFactsFromReport_NotProducingIsWaitable(t *testing.T) {
	ns := node.NodeSet{Nodes: []node.Node{{Index: 1, PID: 10}}}
	rep := health.Report{
		Producing: false,
		Nodes:     []health.NodeInfo{{Index: 1, OK: true, BlockNumber: 0}},
	}
	f := factsFromReport(rep, ns)[0]
	if v := nodemonitor.Classify(f).Verdict; v != nodemonitor.Waitable {
		t.Fatalf("verdict = %s, want WAITABLE when alive but not producing", v)
	}
}

func TestClampCount(t *testing.T) {
	if got := clampCount(5); got != 5 {
		t.Errorf("clampCount(5) = %d, want 5", got)
	}
	if got := clampCount(1 << 40); got != (1<<31)-1 {
		t.Errorf("clampCount(large) = %d, want MaxInt32", got)
	}
}
