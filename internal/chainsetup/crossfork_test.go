package chainsetup

import (
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/core/statemachine"
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
//
// Asked of the first moment, because that is where the crossing begins: a
// refusal that arrived later would already have moved a network.
func TestCrossFork_ANetworkWithNoForkIsRefusedByName(t *testing.T) {
	w := crossWorkspace(t)
	w.state.Fork = nil
	if _, err := w.ForkStanding(); err == nil {
		t.Fatal("a network that crosses no fork accepted the step")
	}
}

// TestCrossFork_AForkWithNoSuccessorIsRefused: the handover needs somebody to
// hand to, and a fork naming a binary no node runs is a declaration that cannot
// happen rather than a wait that will not end.
func TestCrossFork_AForkWithNoSuccessorIsRefused(t *testing.T) {
	w := crossWorkspace(t)
	w.state.Nodes = []node.Record{{Index: 1, Role: string(node.RoleBP)}}
	if _, err := w.ForkStanding(); err == nil {
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
	standing, err := w.ForkStanding()
	if err != nil {
		t.Fatalf("ForkStanding on a crossed network: %v", err)
	}
	if !standing.Crossed {
		t.Fatal("a network whose every successor produces read as not yet across")
	}
	detail, err := w.ReportAlreadyCrossed()
	if err != nil {
		t.Fatalf("ReportAlreadyCrossed: %v", err)
	}
	if detail == "" {
		t.Fatal("the step reported nothing")
	}

	// One left behind is not "already crossed".
	w.state.Nodes[3].Role = string(node.RoleEN)
	if w.alreadyCrossed(*w.state.Fork, w.forkSuccessors(*w.state.Fork)) {
		t.Error("a network with one successor still an endpoint read as fully crossed")
	}
}

// TestCrossingFork_TheMomentsAreStatesAndTheyAreWalkedInOrder.
//
// The three moments were a list the verb appended to as it went. What a reader
// wants from a stopped crossing is which of them it got through, and a list
// kept by hand could not answer it for the one case that walks none of them:
// an already-crossed network was reported as having passed all three. So the
// moments are states, and where the machine is is the answer.
func TestCrossingFork_TheMomentsAreStatesAndTheyAreWalkedInOrder(t *testing.T) {
	mg := newTestManager(t)
	x := mg.crossingFork
	for _, c := range []struct {
		name string
		msg  statemachine.Message
		want statemachine.StateName
	}{
		{"a network still short of the fork begins at the boundary",
			forkStandingRead{Crossed: false}, nameChainOpCrossForkAwaitBoundary},
		{"the boundary reached hands production over",
			forkBoundaryReached{Head: 99}, nameChainOpCrossForkHandOver},
		{"the hand-over done is the crossing",
			productionHandedOver{}, nameChainOpCrossForkConfirm},
		{"a network already across walks none of it",
			forkStandingRead{Crossed: true}, nameChainOpCrossForkConfirm},
	} {
		t.Run(c.name, func(t *testing.T) {
			got, ok := x.nextMoment(c.msg)
			if !ok {
				t.Fatalf("%T is not one of this crossing's messages", c.msg)
			}
			if got.Name() != c.want {
				t.Errorf("%T moves to %s, want %s", c.msg, got.Name(), c.want)
			}
		})
	}
	// The head the boundary reported is what the last moment writes its
	// sentence from, so it is kept rather than read again from a chain that has
	// moved on since.
	if x.head != 99 {
		t.Errorf("the crossing kept head %d, want the one the boundary reported", x.head)
	}
	if !x.already {
		t.Error("a crossing told the network was already across did not remember it")
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

// restartWorkspace is a network composed to cross its OWN chain's fork: every
// node runs the pre-fork build, and at the fork every one of them moves to the
// post-fork one.
func restartWorkspace(t *testing.T) *Workspace {
	t.Helper()
	w := crossWorkspace(t)
	w.state.Fork = &GenesisFork{Name: "boho", At: 200, Binary: "postfork", Restart: true}
	w.state.Binaries = map[string]string{"default": "/b/gstable", "postfork": "/b/gstable-next"}
	w.state.Nodes = []node.Record{
		{Index: 1, Role: string(node.RoleBP), PID: 11},
		{Index: 2, Role: string(node.RoleBP), PID: 12},
		{Index: 3, Role: string(node.RoleBP), PID: 13},
	}
	return w
}

// TestForkSuccessors_ARestartMovesEveryNode.
//
// A handover names the nodes already running the post-fork build. A restart
// cannot: none of them is running it yet, which is the whole point. So every
// node is on the list.
func TestForkSuccessors_ARestartMovesEveryNode(t *testing.T) {
	w := restartWorkspace(t)
	if got := w.forkSuccessors(*w.state.Fork); len(got) != len(w.state.Nodes) {
		t.Fatalf("successors = %v, want every node", got)
	}
}

// TestAlreadyCrossed_ARestartIsReadFromTheBuildNotTheRole.
//
// Crossing changes what a node IS, and the two shapes change different things:
// the work it does on a handover, the build it runs on a restart. Reading the
// role on a restart would report a crossed network as uncrossed forever, since
// a restart leaves every role exactly where it was.
func TestAlreadyCrossed_ARestartIsReadFromTheBuildNotTheRole(t *testing.T) {
	w := restartWorkspace(t)
	f := *w.state.Fork
	if w.alreadyCrossed(f, w.forkSuccessors(f)) {
		t.Error("a network still on the pre-fork build read as crossed")
	}
	for i := range w.state.Nodes {
		w.state.Nodes[i].Binary = f.Binary
	}
	if !w.alreadyCrossed(f, w.forkSuccessors(f)) {
		t.Error("a network already on the post-fork build read as uncrossed")
	}
	// One left behind is not crossed: it would meet the fork on the old build.
	w.state.Nodes[1].Binary = ""
	if w.alreadyCrossed(f, w.forkSuccessors(f)) {
		t.Error("a network with one node still on the pre-fork build read as crossed")
	}
}

// TestForkObserver_ARestartReadsAnyRunningNode: before a restart every node is
// on the pre-fork side, so the first one that answers is the one to read.
func TestForkObserver_ARestartReadsAnyRunningNode(t *testing.T) {
	w := restartWorkspace(t)
	obs, ok := w.forkObserver(*w.state.Fork)
	if !ok || obs.Index != 1 {
		t.Fatalf("observer = %+v (%v), want the first running node", obs, ok)
	}
	for i := range w.state.Nodes {
		w.state.Nodes[i].PID = 0
	}
	if _, ok := w.forkObserver(*w.state.Fork); ok {
		t.Error("a stopped network offered a node to read the head from")
	}
}
