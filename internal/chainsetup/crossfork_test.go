package chainsetup

import (
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// crossWorkspace is a composed network standing at its fork: one pre-fork
// producer and three successors, none of which produces yet.
func crossWorkspace(t *testing.T) *Workspace {
	t.Helper()
	clock := func() time.Time { return time.Unix(0, 0).UTC() }
	comp, err := session.OpenComposition(t.TempDir(), clock)
	if err != nil {
		t.Fatal(err)
	}
	return &Workspace{comp: comp, now: clock, state: State{
		Steps: map[string]Step{},
		Fork:  &GenesisFork{Name: "croissant", At: 100, Binary: "to"},
		Nodes: []node.Record{
			{Index: 1, Role: string(node.RoleBP), PID: 11},
			{Index: 2, Role: string(node.RoleEN), Binary: "to", PID: 12},
			{Index: 3, Role: string(node.RoleEN), Binary: "to", PID: 13},
			{Index: 4, Role: string(node.RoleEN), Binary: "to", PID: 14},
		},
	}}
}

// TestForkSuccessors_TheBuildDecidesWhoTakesOver.
//
// Not the role. Before the fork these nodes are endpoints and after it they
// produce, so the role cannot be the answer at any single moment — the build
// each node runs is, which is the same question binaryFor, genesisFor and
// pluginFor each answer.
func TestForkSuccessors_TheBuildDecidesWhoTakesOver(t *testing.T) {
	w := crossWorkspace(t)
	got := w.forkSuccessors(*w.state.Fork)
	if len(got) != 3 {
		t.Fatalf("successors = %v, want the three nodes running the post-fork build", got)
	}
	for _, i := range got {
		if w.state.Nodes[i].Binary != "to" {
			t.Errorf("position %d runs %q, which is not the fork's binary", i, w.state.Nodes[i].Binary)
		}
	}
}

// TestCrossFork_ANetworkWithNoForkIsRefusedByName: the step is only meaningful
// on a network composed to cross one, and saying so beats waiting out a timeout
// on a chain that was never going to stop.
func TestCrossFork_ANetworkWithNoForkIsRefusedByName(t *testing.T) {
	w := crossWorkspace(t)
	w.state.Fork = nil
	if _, err := w.CrossFork(t.Context(), CrossForkOpts{}); err == nil {
		t.Fatal("a network that crosses no fork accepted the step")
	}
}

// TestCrossFork_AForkWithNoSuccessorIsRefused: the handover needs somebody to
// hand to, and a fork naming a binary no node runs is a declaration that cannot
// happen rather than a wait that will not end.
func TestCrossFork_AForkWithNoSuccessorIsRefused(t *testing.T) {
	w := crossWorkspace(t)
	w.state.Nodes = []node.Record{{Index: 1, Role: string(node.RoleBP)}}
	if _, err := w.CrossFork(t.Context(), CrossForkOpts{}); err == nil {
		t.Fatal("a fork with nobody to hand over to was accepted")
	}
}

// TestCrossFork_ANetworkAlreadyAcrossIsLeftAlone.
//
// A case may name the step on a composition that crossed the fork on its own.
// Bouncing a working chain there would be worse than doing nothing, so the step
// reports what it found. The check is every successor, not any: a network where
// only some produce still has the rest to tell, and the quorum the genesis
// declares is not reached until they are.
func TestCrossFork_ANetworkAlreadyAcrossIsLeftAlone(t *testing.T) {
	w := crossWorkspace(t)
	for i := range w.state.Nodes {
		if w.state.Nodes[i].Binary == "to" {
			w.state.Nodes[i].Role = string(node.RoleBP)
		}
	}
	detail, err := w.CrossFork(t.Context(), CrossForkOpts{})
	if err != nil {
		t.Fatalf("CrossFork on a crossed network: %v", err)
	}
	if detail == "" {
		t.Fatal("the step reported nothing")
	}

	// One left behind is not "already crossed".
	w.state.Nodes[3].Role = string(node.RoleEN)
	if w.alreadyCrossed(w.forkSuccessors(*w.state.Fork)) {
		t.Error("a network with one successor still an endpoint read as fully crossed")
	}
}

// TestForkObserver_TheHeadIsReadFromThePreForkSide.
//
// The pre-fork build refuses to seal the fork block, so its head rises to At-1
// and stops there: that is the signal. A successor's head says the same number
// while it is still syncing and something else entirely once it is not, so
// reading it would confuse "not there yet" with "here".
func TestForkObserver_TheHeadIsReadFromThePreForkSide(t *testing.T) {
	w := crossWorkspace(t)
	obs, ok := w.forkObserver(*w.state.Fork)
	if !ok {
		t.Fatal("no observer on a network that has a pre-fork producer")
	}
	if obs.Index != 1 {
		t.Errorf("observer is node%d, want the node running the pre-fork build", obs.Index)
	}

	// A stopped pre-fork node is not an observer: it answers nothing.
	w.state.Nodes[0].PID = 0
	if _, ok := w.forkObserver(*w.state.Fork); ok {
		t.Error("a stopped node was chosen to read the fork boundary from")
	}
}
