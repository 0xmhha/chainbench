package testengine_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// recorded builds the record a composed network leaves behind.
func recorded(nodes ...node.Record) chainsetup.State {
	return chainsetup.State{Chain: "stablenet", KeysDir: "presets/keys", Nodes: nodes}
}

func bp(i int, args ...string) node.Record {
	return node.Record{Index: i, Label: "node" + itoa(i), Role: string(node.RoleBP), Args: args}
}

func en(i int, args ...string) node.Record {
	return node.Record{Index: i, Label: "node" + itoa(i), Role: string(node.RoleEN), Args: args}
}

func itoa(i int) string { return string(rune('0' + i)) }

// planFor builds a plan asking for what recorded() would satisfy.
func planFor() testengine.ComposePlan {
	p := testengine.ComposePlan{Chain: "stablenet"}
	p.Keys.Dir = "presets/keys"
	p.Nodes.BP, p.Nodes.EN = 2, 1
	return p
}

// TestVerifyLaunched_AcceptsTheNetworkItAskedFor.
func TestVerifyLaunched_AcceptsTheNetworkItAskedFor(t *testing.T) {
	got := testengine.VerifyLaunched(planFor(), recorded(bp(1), bp(2), en(3)))
	if len(got) != 0 {
		t.Fatalf("a matching network reported %v", got)
	}
}

// TestVerifyLaunched_CatchesAChainThatIsNotTheOnePlanned is the coarsest
// mismatch and the one a reader would never suspect: every step reports
// success, and the tests answer about another chain.
func TestVerifyLaunched_CatchesAChainThatIsNotTheOnePlanned(t *testing.T) {
	st := recorded(bp(1), bp(2), en(3))
	st.Chain = "wbft"
	got := testengine.VerifyLaunched(planFor(), st)
	if len(got) != 1 || !strings.Contains(got[0].String(), "wbft") {
		t.Fatalf("mismatches = %v", got)
	}
}

// TestVerifyLaunched_CatchesAShapeThatIsNotThePlanned.
func TestVerifyLaunched_CatchesAShapeThatIsNotThePlanned(t *testing.T) {
	got := testengine.VerifyLaunched(planFor(), recorded(bp(1), en(3)))
	if len(got) != 1 {
		t.Fatalf("one bp short must be one mismatch, got %v", got)
	}
	if !strings.Contains(got[0].String(), "2 bp") {
		t.Errorf("the mismatch must say what it wanted: %s", got[0])
	}
}

// TestVerifyLaunched_AutoSizedBPIsAFloor: a plan whose bp count fills from the
// server set states a minimum, so a bigger table is what it asked for and a
// smaller one is not.
func TestVerifyLaunched_AutoSizedBPIsAFloor(t *testing.T) {
	p := planFor()
	p.Nodes.AutoSize = true
	if got := testengine.VerifyLaunched(p, recorded(bp(1), bp(2), bp(4), en(3))); len(got) != 0 {
		t.Errorf("more bp than the floor is not a mismatch: %v", got)
	}
	if got := testengine.VerifyLaunched(p, recorded(bp(1), en(3))); len(got) != 1 {
		t.Errorf("fewer bp than the floor is: %v", got)
	}
}

// TestVerifyLaunched_CatchesAKnobThatNeverReachedArgv is why this exists.
//
// The command line has the last word, so a knob a declaration asked for can be
// merged correctly and still not be on the node. Nothing else in this package
// would notice: the plan says it was asked for and the run says every step
// succeeded.
func TestVerifyLaunched_CatchesAKnobThatNeverReachedArgv(t *testing.T) {
	p := planFor()
	p.Launch = map[string][]testengine.PlanKnob{
		node.ScopeAll: {{Knob: "nodiscover", From: testengine.SourceCommand}},
		"bp":          {{Knob: "mine", From: testengine.SourceDeclaration}},
	}

	full := recorded(
		bp(1, "--nodiscover", "--mine"),
		bp(2, "--nodiscover", "--mine"),
		en(3, "--nodiscover"),
	)
	if got := testengine.VerifyLaunched(p, full); len(got) != 0 {
		t.Fatalf("every knob is present: %v", got)
	}

	// node2 lost --mine, and the bp scope covers it.
	missing := recorded(
		bp(1, "--nodiscover", "--mine"),
		bp(2, "--nodiscover"),
		en(3, "--nodiscover"),
	)
	got := testengine.VerifyLaunched(p, missing)
	if len(got) != 1 {
		t.Fatalf("mismatches = %v", got)
	}
	if !strings.Contains(got[0].String(), "node2") || !strings.Contains(got[0].String(), "mine") {
		t.Errorf("the mismatch must name the node and the knob: %s", got[0])
	}
	// Whoever reads this has to go change something, so it says where to go.
	if !strings.Contains(got[0].String(), string(testengine.SourceDeclaration)) {
		t.Errorf("the mismatch must name who asked for the knob: %s", got[0])
	}
}

// TestVerifyLaunched_AKnobIsFoundByItsLastSegment: a declaration writes a knob
// as a dot path and the binary takes a dotted flag; neither spelling is the
// other, and the name they share is the last segment.
func TestVerifyLaunched_AKnobIsFoundByItsLastSegment(t *testing.T) {
	p := planFor()
	p.Launch = map[string][]testengine.PlanKnob{node.ScopeAll: {
		{Knob: "chain.networkid=8283", From: testengine.SourceCommand},
		{Knob: "rpc.allow-unprotected-txs", From: testengine.SourceDeclaration},
	}}

	st := recorded(
		bp(1, "--networkid", "8283", "--rpc.allow-unprotected-txs"),
		bp(2, "--networkid", "8283", "--rpc.allow-unprotected-txs"),
		en(3, "--networkid", "8283", "--rpc.allow-unprotected-txs"),
	)
	if got := testengine.VerifyLaunched(p, st); len(got) != 0 {
		t.Fatalf("both spellings must match: %v", got)
	}
}

// TestVerifyLaunched_CatchesAWorkspaceComposedWithAnotherBinary is the reuse
// case. A workspace is composed once and run against many times, so the plan is
// this run's and the record may be an earlier run's. A network built from
// another binary answers every RPC the suite asks and answers it about a
// different program.
func TestVerifyLaunched_CatchesAWorkspaceComposedWithAnotherBinary(t *testing.T) {
	p := planFor()
	p.Binary = "/b/gstable"
	p.From = map[testengine.PlanField]testengine.PlanSource{
		testengine.FieldBinary: testengine.SourceCommand,
	}

	st := recorded(bp(1), bp(2), en(3))
	st.Binary = "/b/gstable"
	if got := testengine.VerifyLaunched(p, st); len(got) != 0 {
		t.Fatalf("the same binary must not be a mismatch: %v", got)
	}

	st.Binary = "/b/gwemix"
	got := testengine.VerifyLaunched(p, st)
	if len(got) != 1 {
		t.Fatalf("mismatches = %v", got)
	}
	if !strings.Contains(got[0].String(), "gwemix") || !strings.Contains(got[0].String(), "gstable") {
		t.Errorf("the mismatch must name both binaries: %s", got[0])
	}
	if !strings.Contains(got[0].String(), string(testengine.SourceCommand)) {
		t.Errorf("the mismatch must name who asked for the binary: %s", got[0])
	}
}

// TestVerifyLaunched_CatchesNodesOnAnotherMachine is the same hole one field
// over: the declaration places the network, and a workspace composed under a
// different placement holds nodes somewhere else entirely.
func TestVerifyLaunched_CatchesNodesOnAnotherMachine(t *testing.T) {
	p := planFor()
	p.Placement = resource.Spec{Host: "a.example", User: "ops", DataRoot: "/data/net1"}

	st := recorded(bp(1), bp(2), en(3))
	st.Target = p.Placement
	if got := testengine.VerifyLaunched(p, st); len(got) != 0 {
		t.Fatalf("the same placement must not be a mismatch: %v", got)
	}

	st.Target = resource.Spec{Host: "b.example", User: "ops", DataRoot: "/data/net1"}
	got := testengine.VerifyLaunched(p, st)
	if len(got) != 1 || !strings.Contains(got[0].String(), "b.example") {
		t.Fatalf("mismatches = %v", got)
	}
}

// TestStopAfterFailedSetup covers the two directions of taking a network down
// when setting it up failed.
//
// Setting up is not all-or-nothing: the nodes launch and then the readiness
// gate refuses what came up. The run then reported the failure and left four
// nodes holding their ports, so the next run could not compose at all. The
// second case is the one that keeps the first honest — an operator who passed
// --keep-up wants to look at exactly this network.
func TestStopAfterFailedSetup(t *testing.T) {
	setupErr := errors.New("network not ready to test")

	for _, tc := range []struct {
		name       string
		keepUp     bool
		hasNet     bool
		wantStops  int
		wantErrHas string
	}{
		{"a failed setup takes the network down", false, true, 1, "not ready"},
		{"keep-up leaves it up to be looked at", true, true, 0, "not ready"},
		{"nothing came up, nothing to take down", false, false, 0, "not ready"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stops := 0
			net := testengine.ComposedForTest(nil)
			if tc.hasNet {
				net = testengine.ComposedForTest(func(context.Context) error {
					stops++
					return nil
				})
			}
			err := testengine.AfterFailedSetupForTest(context.Background(), net, tc.keepUp, setupErr)
			if stops != tc.wantStops {
				t.Errorf("teardown ran %d time(s), want %d", stops, tc.wantStops)
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErrHas) {
				t.Errorf("the setup error must survive: %v", err)
			}
		})
	}
}

// TestStopAfterFailedSetup_SaysWhenItCouldNotStop: a network that will not go
// down is worse news than the setup failure, and the operator has to be told
// both — the one they were waiting for, and the one that will block their next
// run.
func TestStopAfterFailedSetup_SaysWhenItCouldNotStop(t *testing.T) {
	net := testengine.ComposedForTest(func(context.Context) error {
		return errors.New("node2 is still present after SIGKILL")
	})
	err := testengine.AfterFailedSetupForTest(context.Background(), net, false, errors.New("not ready"))
	if err == nil || !strings.Contains(err.Error(), "not ready") || !strings.Contains(err.Error(), "SIGKILL") {
		t.Fatalf("both failures must be named: %v", err)
	}
}

// TestVerifyLaunched_APlacementIsHeldToWhatItNamed.
//
// A placement is what the ENV declared; a machine chosen on the command line is
// a separate field. So a run placed with --server or --all-servers has a
// placement naming no machine, while the record always names the one the
// workspace composed on — and a spread network cannot name one at all.
//
// Measured: a hardfork composed across the docker server set passed every step
// including start, then was refused with "asked for nodes on local
// /data/chainbench, launched with nodes on server server1:/data/chainbench".
// Two ways of saying the same placement.
func TestVerifyLaunched_APlacementIsHeldToWhatItNamed(t *testing.T) {
	// Named no machine, only a data root: the server the record kept is not a
	// contradiction, and the data root is what was asked.
	p := planFor()
	p.Placement = resource.Spec{DataRoot: "/data/chainbench"}
	st := recorded(bp(1), bp(2), en(3))
	st.Target = resource.Spec{Server: "server1", DataRoot: "/data/chainbench"}
	if got := testengine.VerifyLaunched(p, st); len(got) != 0 {
		t.Errorf("a spread placement was reported as a mismatch: %v", got)
	}

	// The same placement against another data root is still a mismatch.
	st.Target = resource.Spec{Server: "server1", DataRoot: "/elsewhere"}
	if got := testengine.VerifyLaunched(p, st); len(got) == 0 {
		t.Error("a network composed under another data root passed")
	}

	// A placement that DOES name a machine is still held to that machine.
	p.Placement = resource.Spec{Server: "server9", DataRoot: "/data/chainbench"}
	st.Target = resource.Spec{Server: "server1", DataRoot: "/data/chainbench"}
	if got := testengine.VerifyLaunched(p, st); len(got) == 0 {
		t.Error("a network composed on a machine the env did not name passed")
	}
}
