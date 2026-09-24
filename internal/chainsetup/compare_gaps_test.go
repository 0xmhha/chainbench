package chainsetup

import (
	"context"
	"slices"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/preflight"
	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// These hold the three defects the 2026-09-24 rerun found (analyses 19, design
// state-machine-06 §10 E1–E3). Each was written first and failed on the code
// it describes: the comparison's two follow-up messages were handled only by a
// sibling of the state that sent them, so they were dropped, the machine
// stalled mid-comparison, and ComposeComparing still returned nil.

// verdictManager is a manager whose comparison answers v without a network.
func verdictManager(t *testing.T, v preflight.Verdict) (*Manager, ChainUpIn) {
	t.Helper()
	mg := newTestManager(t)
	mg.compare = func(context.Context, Deps, ChainUpIn) preflight.Decision {
		return preflight.Decision{Verdict: v, Reasons: []string{"injected by the test"}}
	}
	in := withStage(upRequest(t), UpDeploy)
	in.DataDir = mg.ws.Dir()
	return mg, in
}

// at is where the machine came to rest.
func at(mg *Manager) statemachine.StateName { return mg.m.Current().Name() }

// E1: a network that differs is stopped and then composed again. The stop's
// report has to reach something that starts the composition.
func TestComposeComparing_RebuildAllComposesAgain(t *testing.T) {
	mg, in := verdictManager(t, preflight.RebuildAll)
	err := mg.ComposeComparing(context.Background(), in)
	if err != nil {
		t.Fatalf("rebuild-all failed: %v", err)
	}
	if got := entered(mg); !slices.Contains(got, string(nameChainOpenWorkspace)) {
		t.Errorf("after stopping, the run went through %v and never composed again", got)
	}
	if got := at(mg); got != nameChainBuildUpStoppedAtStep {
		t.Errorf("rebuild-all stopped in %s, want %s (the request stops at deploy)", got, nameChainBuildUpStoppedAtStep)
	}
}

// E2: nodes that differ come back and the network is ready. The restart's
// report has to reach something that ends the comparison.
func TestComposeComparing_RebuildNodesEndsReady(t *testing.T) {
	mg, in := verdictManager(t, preflight.RebuildNodes)
	if err := mg.ComposeComparing(context.Background(), in); err != nil {
		t.Fatalf("rebuild-nodes failed: %v", err)
	}
	if got := at(mg); got != nameChainReady {
		t.Errorf("rebuild-nodes stopped in %s, want %s", got, nameChainReady)
	}
}

// E3: an entry point that returns without error has put the machine in one of
// its terminal states. Returning nil from anywhere else reports a stall as a
// success, which is what sent the rerun on to a readiness gate over a network
// that was never launched.
func TestComposeComparing_NilOnlyAtATerminalState(t *testing.T) {
	for _, v := range []preflight.Verdict{preflight.Reuse, preflight.RebuildNodes, preflight.RebuildAll, preflight.Compose} {
		mg, in := verdictManager(t, v)
		if err := mg.ComposeComparing(context.Background(), in); err != nil {
			continue
		}
		if got := at(mg); !slices.Contains([]statemachine.StateName{nameChainReady, nameChainBuildUpStoppedAtStep, nameChainFailed}, got) {
			t.Errorf("%s: ComposeComparing returned nil while the machine was in %s", v, got)
		}
	}
}

// TestComposeComparing_RelaunchSkipsTheSetup: the network is the one wanted and
// none of its nodes runs, so it goes straight to launching its nodes — nothing
// about it is built again (design-v3 state-machine-06 R3).
func TestComposeComparing_RelaunchSkipsTheSetup(t *testing.T) {
	mg, in := verdictManager(t, preflight.Relaunch)
	in.Stage = ""
	_ = mg.ComposeComparing(context.Background(), in) // the launch itself has no binary here
	got := entered(mg)
	for _, want := range []statemachine.StateName{nameChainCompareNetworkStopped, nameChainLaunchNodes} {
		if !slices.Contains(got, string(want)) {
			t.Errorf("a relaunch went through %v and never entered %s", got, want)
		}
	}
	for _, skipped := range []statemachine.StateName{nameChainOpenWorkspace, nameChainBuildGenesis, nameChainInitNodes} {
		if slices.Contains(got, string(skipped)) {
			t.Errorf("a relaunch rebuilt its setup: it entered %s (%v)", skipped, got)
		}
	}
}

// TestOperate_StopRestsInStopped: an operation's parent says where the network
// stands afterwards, and a stopped network is not ready (S2).
func TestOperate_StopRestsInStopped(t *testing.T) {
	mg := newTestManager(t)
	if err := mg.Compose(context.Background(), withStage(upRequest(t), UpDeploy), ""); err != nil {
		t.Fatal(err)
	}
	op := NewManager(Deps{}, mg.ws)
	if _, err := op.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if got := at(op); got != nameChainStopped {
		t.Errorf("a stop rested in %s, want %s", got, nameChainStopped)
	}
}
