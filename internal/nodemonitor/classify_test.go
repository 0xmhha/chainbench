package nodemonitor_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/nodemonitor"
)

// ready is a fully-fit wanted node; each case perturbs one axis.
func ready() nodemonitor.Facts {
	return nodemonitor.Facts{
		Node: 1, Label: "node1", Wanted: true,
		WantChainID: 1000, WantPeers: 3, WantParticipate: true,
		PIDAlive: true, RPCUp: true, ChainID: 1000, Height: 42,
		Advancing: true, Syncing: false, Peers: 3, Participating: true,
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		mut  func(*nodemonitor.Facts)
		want nodemonitor.Verdict
	}{
		{"fully ready", func(*nodemonitor.Facts) {}, nodemonitor.Ready},
		{"not wanted is ready", func(f *nodemonitor.Facts) { f.Wanted = false; f.PIDAlive = false }, nodemonitor.Ready},

		// FATAL — destructive remedy needed.
		{"wrong chain id", func(f *nodemonitor.Facts) { f.ChainID = 999 }, nodemonitor.Fatal},
		{"forked", func(f *nodemonitor.Facts) { f.Forked = true }, nodemonitor.Fatal},
		{"etcd stale", func(f *nodemonitor.Facts) { f.Failure = process.EtcdStale }, nodemonitor.Fatal},
		{"quorum lost", func(f *nodemonitor.Facts) { f.Failure = process.QuorumLost }, nodemonitor.Fatal},

		// RESTARTABLE — a fresh start may clear it.
		{"process dead", func(f *nodemonitor.Facts) { f.PIDAlive = false }, nodemonitor.Restartable},
		{"rpc unready flag", func(f *nodemonitor.Facts) { f.Failure = process.RPCUnready }, nodemonitor.Restartable},
		{"etcd join failed", func(f *nodemonitor.Facts) { f.Failure = process.EtcdJoinFailed }, nodemonitor.Restartable},
		{"fork not crossed", func(f *nodemonitor.Facts) { f.Failure = process.ForkNotCrossed }, nodemonitor.Restartable},
		{"rpc down", func(f *nodemonitor.Facts) { f.RPCUp = false }, nodemonitor.Restartable},

		// WAITABLE — alive and reachable, not there yet.
		{"syncing", func(f *nodemonitor.Facts) { f.Syncing = true }, nodemonitor.Waitable},
		{"not advancing", func(f *nodemonitor.Facts) { f.Advancing = false }, nodemonitor.Waitable},
		{"few peers", func(f *nodemonitor.Facts) { f.Peers = 1 }, nodemonitor.Waitable},
		{"not participating", func(f *nodemonitor.Facts) { f.Participating = false }, nodemonitor.Waitable},

		// A dead process is RESTARTABLE even though it is also not advancing:
		// the worse verdict wins.
		{"dead beats waitable", func(f *nodemonitor.Facts) { f.PIDAlive = false; f.Advancing = false }, nodemonitor.Restartable},
		// A wrong chain id is FATAL even with a dead-looking RPC path: FATAL wins.
		{"fatal beats restartable", func(f *nodemonitor.Facts) { f.ChainID = 7; f.Failure = process.RPCUnready }, nodemonitor.Fatal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := ready()
			tc.mut(&f)
			got := nodemonitor.Classify(f)
			if got.Verdict != tc.want {
				t.Fatalf("verdict = %s, want %s (reasons: %v)", got.Verdict, tc.want, got.Reasons)
			}
			if got.Verdict != nodemonitor.Ready && len(got.Reasons) == 0 {
				t.Error("a non-ready verdict must carry a reason")
			}
		})
	}
}

// TestClassify_PeersUnconstrained: WantPeers 0 means the peer count does not
// gate readiness (a single-node or don't-care topology).
func TestClassify_PeersUnconstrained(t *testing.T) {
	f := ready()
	f.WantPeers = 0
	f.Peers = 0
	if got := nodemonitor.Classify(f); got.Verdict != nodemonitor.Ready {
		t.Fatalf("verdict = %s, want READY when peers unconstrained", got.Verdict)
	}
}
