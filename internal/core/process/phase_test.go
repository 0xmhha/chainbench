package process

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

func phasePlan(n int) Plan {
	p := Plan{DataRoot: "/d"}
	for i := 1; i <= n; i++ {
		p.Nodes = append(p.Nodes, NodeSpec{Index: i, Role: node.RoleBP, Host: "127.0.0.1"})
	}
	return p
}

// launchRecorder answers a launch with the nodes it was asked for, recording
// the selection so a test can see what each phase actually started.
func launchRecorder(got *[][]int) func(context.Context, Plan, []int) (Result, error) {
	return func(_ context.Context, plan Plan, nodes []int) (Result, error) {
		*got = append(*got, nodes)
		want := nodes
		if len(want) == 0 {
			for _, s := range plan.Nodes {
				want = append(want, s.Index)
			}
		}
		var res Result
		for _, i := range want {
			res.Nodes.Nodes = append(res.Nodes.Nodes, node.Node{Index: i, Role: node.RoleBP})
		}
		return res, nil
	}
}

// TestBringUp_NoPhasesLaunchesEverythingAtOnce pins the behaviour every wbft
// network has: one launch of the whole plan, which is what the supervisor did
// before phases existed.
func TestBringUp_NoPhasesLaunchesEverythingAtOnce(t *testing.T) {
	var launches [][]int
	s := NewLauncher(Deps{
		Launch:     launchRecorder(&launches),
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		Procman:    New(),
	})
	ns, _, err := s.BringUp(context.Background(), phasePlan(4), Options{})
	if err != nil {
		t.Fatalf("BringUp: %v", err)
	}
	if len(launches) != 1 || len(launches[0]) != 0 {
		t.Fatalf("launches = %v, want a single whole-plan launch", launches)
	}
	if len(ns.Nodes) != 4 {
		t.Fatalf("brought up %d nodes, want 4", len(ns.Nodes))
	}
}

// TestBringUp_PhasesLaunchInOrderAndRunActionsBetween is the sequence a wemix
// network needs: the producer alone, its bootstrap, then everyone else. The
// cluster only forms while the producer is alone, so the order is the feature.
func TestBringUp_PhasesLaunchInOrderAndRunActionsBetween(t *testing.T) {
	var launches [][]int
	var actions []string
	var actionSawNodes int

	s := NewLauncher(Deps{
		Launch: launchRecorder(&launches),
		Action: func(_ context.Context, name string, _ Plan, on node.Node) error {
			actions = append(actions, name)
			actionSawNodes = on.Index
			return nil
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		Procman:    New(),
	})
	ns, _, err := s.BringUp(context.Background(), phasePlan(4), Options{Phases: []registry.Phase{
		{Name: "boot", Nodes: []int{1}, Actions: []string{"deploy-governance", "etcd-init"}},
		{Name: "rest", Nodes: []int{2, 3, 4}},
	}})
	if err != nil {
		t.Fatalf("BringUp: %v", err)
	}
	if len(launches) != 2 || len(launches[0]) != 1 || launches[0][0] != 1 || len(launches[1]) != 3 {
		t.Fatalf("launches = %v, want the producer then the rest", launches)
	}
	if strings.Join(actions, ",") != "deploy-governance,etcd-init" {
		t.Fatalf("actions = %v, want both in order", actions)
	}
	// The bootstrap runs on the node its phase started, not on some later one.
	if actionSawNodes != 1 {
		t.Fatalf("action ran on node%d, want the producer", actionSawNodes)
	}
	// Health is judged against the whole network, not against the last phase.
	if len(ns.Nodes) != 4 {
		t.Fatalf("brought up %d nodes, want all 4", len(ns.Nodes))
	}
}

// TestBringUp_UnwiredActionIsAnError: a bootstrap that is quietly skipped
// produces a network that starts and then never produces a block, which is the
// failure this whole family's bring-up exists to avoid. Same contract as
// LeaderGate and SwapBinary.
func TestBringUp_UnwiredActionIsAnError(t *testing.T) {
	var launches [][]int
	s := NewLauncher(Deps{
		Launch:     launchRecorder(&launches),
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		Procman:    New(),
	})
	_, _, err := s.BringUp(context.Background(), phasePlan(2), Options{Phases: []registry.Phase{
		{Name: "boot", Nodes: []int{1}, Actions: []string{"deploy-governance"}},
	}})
	if err == nil {
		t.Fatal("an action with no executor must fail the bring-up")
	}
	if !strings.Contains(err.Error(), "deploy-governance") {
		t.Fatalf("error %q should name the action", err)
	}
}

func TestBringUp_ActionFailureStopsTheBringUp(t *testing.T) {
	var launches [][]int
	s := NewLauncher(Deps{
		Launch:     launchRecorder(&launches),
		Action:     func(context.Context, string, Plan, node.Node) error { return errors.New("etcd cluster empty") },
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		Procman:    New(),
	})
	_, _, err := s.BringUp(context.Background(), phasePlan(3), Options{Phases: []registry.Phase{
		{Name: "boot", Nodes: []int{1}, Actions: []string{"etcd-init"}},
		{Name: "rest", Nodes: []int{2, 3}},
	}})
	if err == nil {
		t.Fatal("a failed bootstrap must fail the bring-up")
	}
	if len(launches) != 1 {
		t.Fatalf("the second phase must not launch after a failed action: %v", launches)
	}
}

// TestBringUp_FailedLaunchLeavesNothingRunning: a bring-up that fails part way
// through still started processes, and with phases it always has — the group
// before the failing one is up. Leaving them holds the ports the next attempt
// needs, so the retry fails for a reason unrelated to the first, and an
// operator reading "address already in use" is looking at the wrong problem.
func TestBringUp_FailedLaunchLeavesNothingRunning(t *testing.T) {
	var launches [][]int
	var stopped []node.NodeSet
	s := NewLauncher(Deps{
		Launch: launchRecorder(&launches),
		Action: func(context.Context, string, Plan, node.Node) error {
			return errors.New("etcd cluster empty")
		},
		HealthGate: func(context.Context, node.NodeSet) (Diagnosis, error) { return Diagnosis{OK: true}, nil },
		Procman:    New(),
	})
	impl := s.(*impl)
	impl.teardownHook = func(ns node.NodeSet) { stopped = append(stopped, ns) }

	_, _, err := s.BringUp(context.Background(), phasePlan(3), Options{Phases: []registry.Phase{
		{Name: "boot", Nodes: []int{1}, Actions: []string{"etcd-init"}},
		{Name: "rest", Nodes: []int{2, 3}},
	}})
	if err == nil {
		t.Fatal("the bring-up must fail")
	}
	if len(stopped) != 1 || len(stopped[0].Nodes) != 1 || stopped[0].Nodes[0].Index != 1 {
		t.Fatalf("teardown saw %v, want the one node the boot phase started", stopped)
	}
}
