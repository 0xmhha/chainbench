package supervisor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
)

func TestJoinGap_DerivedFromClusterSize(t *testing.T) {
	cases := []struct {
		size int
		want time.Duration
	}{
		{1, 7 * time.Second},
		{11, 7 * time.Second},
		{12, 11 * time.Second},
		{23, 11 * time.Second},
		{24, 17 * time.Second},
		{41, 17 * time.Second},
		{42, 23 * time.Second},
	}
	for _, tc := range cases {
		if got := JoinGap(tc.size); got != tc.want {
			t.Errorf("JoinGap(%d) = %s, want %s", tc.size, got, tc.want)
		}
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		err  string
		want FailureMode
	}{
		{"join failure", `etcd join failed: not found`, EtcdJoinFailed},
		{"stale cluster info", `cannot fetch cluster info from peer urls`, EtcdStale},
		{"quorum", `not enough validators to reach quorum`, QuorumLost},
		{"fork", `fork block 100 was never crossed`, ForkNotCrossed},
		{"rpc", `dial tcp 127.0.0.1:8600: connection refused`, RPCUnready},
		{"unknown", `something else entirely`, UnknownFailure},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Classify(errors.New(tc.err)); got != tc.want {
				t.Fatalf("Classify(%q) = %s, want %s", tc.err, got, tc.want)
			}
		})
	}
}

func TestFailureMode_ZeroValueIsUnknownNotAnEtcdFailure(t *testing.T) {
	var d Diagnosis
	if d.Mode != UnknownFailure {
		t.Fatalf("zero Diagnosis reports %s; an unset mode must not read as a real cause", d.Mode)
	}
}

func TestBringUp_LeaderGateRunsBeforeTheHealthGate(t *testing.T) {
	var order []string
	sup := New(Deps{
		Launch: func(context.Context, setup.Plan) (LaunchResult, error) {
			order = append(order, "launch")
			return LaunchResult{Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1}}}}, nil
		},
		LeaderGate: func(context.Context, node.NodeSet, time.Duration) (Diagnosis, error) {
			order = append(order, "leader")
			return Diagnosis{OK: true}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) {
			order = append(order, "health")
			return Diagnosis{OK: true}, nil
		},
		Sleep: func(time.Duration) {},
	})
	if _, _, err := sup.BringUp(context.Background(), setup.Plan{}, Options{LeaderGate: true}); err != nil {
		t.Fatalf("BringUp: %v", err)
	}
	want := []string{"launch", "leader", "health"}
	if len(order) != 3 || order[0] != want[0] || order[1] != want[1] || order[2] != want[2] {
		t.Fatalf("order = %v, want %v", order, want)
	}
}

func TestBringUp_LeaderGateSkippedWhenNotRequested(t *testing.T) {
	called := false
	sup := New(Deps{
		Launch: func(context.Context, setup.Plan) (LaunchResult, error) {
			return LaunchResult{Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1}}}}, nil
		},
		LeaderGate: func(context.Context, node.NodeSet, time.Duration) (Diagnosis, error) {
			called = true
			return Diagnosis{OK: true}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		Sleep:      func(time.Duration) {},
	})
	if _, _, err := sup.BringUp(context.Background(), setup.Plan{}, Options{}); err != nil {
		t.Fatalf("BringUp: %v", err)
	}
	if called {
		t.Fatal("the leader gate ran even though Options.LeaderGate was false")
	}
}

func TestBringUp_LeaderGateRequestedButNotWiredIsAnError(t *testing.T) {
	sup := New(Deps{
		Launch: func(context.Context, setup.Plan) (LaunchResult, error) {
			return LaunchResult{Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1}}}}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		Sleep:      func(time.Duration) {},
	})
	_, diag, err := sup.BringUp(context.Background(), setup.Plan{}, Options{LeaderGate: true, MaxAttempts: 1})
	if err == nil {
		t.Fatal("expected an error: a requested gate with no implementation must not pass silently")
	}
	if diag.Mode != EtcdJoinFailed {
		t.Fatalf("mode = %s, want EtcdJoinFailed", diag.Mode)
	}
}

func TestBringUp_AlignJoinGapSizesTheLeaderGateWindow(t *testing.T) {
	var window time.Duration
	nodes := make([]node.Node, 5)
	for i := range nodes {
		nodes[i] = node.Node{Index: i + 1}
	}
	sup := New(Deps{
		Launch: func(context.Context, setup.Plan) (LaunchResult, error) {
			return LaunchResult{Nodes: node.NodeSet{Nodes: nodes}}, nil
		},
		LeaderGate: func(_ context.Context, _ node.NodeSet, w time.Duration) (Diagnosis, error) {
			window = w
			return Diagnosis{OK: true}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		Sleep:      func(time.Duration) {},
	})
	if _, _, err := sup.BringUp(context.Background(), setup.Plan{}, Options{LeaderGate: true, AlignJoinGap: true}); err != nil {
		t.Fatalf("BringUp: %v", err)
	}
	// 5 nodes: gap 7s each, plus one gap of settle margin.
	want := 6 * 7 * time.Second
	if window != want {
		t.Fatalf("leader-gate window = %s, want %s", window, want)
	}
}

func TestBringUp_LaunchErrorIsClassified(t *testing.T) {
	sup := New(Deps{
		Launch: func(context.Context, setup.Plan) (LaunchResult, error) {
			return LaunchResult{}, errors.New("node1: cannot fetch cluster info from peer urls")
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		Sleep:      func(time.Duration) {},
	})
	_, diag, err := sup.BringUp(context.Background(), setup.Plan{}, Options{MaxAttempts: 1})
	if err == nil {
		t.Fatal("expected an error")
	}
	if diag.Mode != EtcdStale {
		t.Fatalf("mode = %s, want EtcdStale (a launch failure must not be reported as RPCUnready)", diag.Mode)
	}
}

func TestBringUp_HealthGateErrorWithoutAModeIsClassified(t *testing.T) {
	sup := New(Deps{
		Launch: func(context.Context, setup.Plan) (LaunchResult, error) {
			return LaunchResult{Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1}}}}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) {
			return Diagnosis{}, errors.New("fork block 100 was never crossed")
		},
		Sleep: func(time.Duration) {},
	})
	_, diag, err := sup.BringUp(context.Background(), setup.Plan{}, Options{MaxAttempts: 1})
	if err == nil {
		t.Fatal("expected an error")
	}
	if diag.Mode != ForkNotCrossed {
		t.Fatalf("mode = %s, want ForkNotCrossed", diag.Mode)
	}
}

func TestBringUp_ExplicitGateModeIsPreserved(t *testing.T) {
	sup := New(Deps{
		Launch: func(context.Context, setup.Plan) (LaunchResult, error) {
			return LaunchResult{Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1}}}}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) {
			return Diagnosis{Mode: QuorumLost, Detail: "3 of 7 validators"}, nil
		},
		Sleep: func(time.Duration) {},
	})
	_, diag, _ := sup.BringUp(context.Background(), setup.Plan{}, Options{MaxAttempts: 1})
	if diag.Mode != QuorumLost || diag.Detail != "3 of 7 validators" {
		t.Fatalf("diagnosis = %+v, want the gate's own classification preserved", diag)
	}
}

func TestBringUp_ForkSwapsRequestedButNotWiredIsAnError(t *testing.T) {
	sup := New(Deps{
		Launch: func(context.Context, setup.Plan) (LaunchResult, error) {
			return LaunchResult{Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1}}}}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		Sleep:      func(time.Duration) {},
	})
	_, diag, err := sup.BringUp(context.Background(), setup.Plan{}, Options{
		MaxAttempts: 1,
		ForkSwaps:   []ForkSwap{{Node: "bp1", Fork: "croissant", ToBinary: "/bin/gwbft", AtBlock: 100}},
	})
	if err == nil {
		t.Fatal("expected an error: a declared fork swap must not be silently dropped")
	}
	if diag.Mode != ForkNotCrossed {
		t.Fatalf("mode = %s, want ForkNotCrossed", diag.Mode)
	}
}

func TestBringUp_ForkSwapsAreScheduledBeforeTheForkBlock(t *testing.T) {
	var swapped []ForkSwap
	sup := New(Deps{
		Launch: func(context.Context, setup.Plan) (LaunchResult, error) {
			return LaunchResult{Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1}}}}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		SwapBinary: func(_ context.Context, _ node.NodeSet, s ForkSwap) error {
			swapped = append(swapped, s)
			return nil
		},
		Sleep: func(time.Duration) {},
	})
	want := ForkSwap{Node: "bp1", Fork: "croissant", ToBinary: "/bin/gwbft", AtBlock: 100}
	if _, _, err := sup.BringUp(context.Background(), setup.Plan{}, Options{ForkSwaps: []ForkSwap{want}}); err != nil {
		t.Fatalf("BringUp: %v", err)
	}
	if len(swapped) != 1 || swapped[0] != want {
		t.Fatalf("swapped = %v, want [%v]", swapped, want)
	}
}

func TestBringUp_ForkSwapFailureIsClassified(t *testing.T) {
	sup := New(Deps{
		Launch: func(context.Context, setup.Plan) (LaunchResult, error) {
			return LaunchResult{Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1}}}}, nil
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		SwapBinary: func(context.Context, node.NodeSet, ForkSwap) error {
			return errors.New("head is already past the fork block")
		},
		Sleep: func(time.Duration) {},
	})
	_, diag, err := sup.BringUp(context.Background(), setup.Plan{}, Options{
		MaxAttempts: 1,
		ForkSwaps:   []ForkSwap{{Node: "bp1", Fork: "croissant", ToBinary: "/bin/gwbft", AtBlock: 100}},
	})
	if err == nil {
		t.Fatal("expected an error")
	}
	if diag.Mode != ForkNotCrossed {
		t.Fatalf("mode = %s, want ForkNotCrossed", diag.Mode)
	}
}
