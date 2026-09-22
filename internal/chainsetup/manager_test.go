package chainsetup

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // the opening stage checks the chain is one we have
	"github.com/0xmhha/chainbench/internal/core/preflight"
	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// The machine's job at this commit is to walk the same nine steps in the same
// order the table walked, stop where it was told, and say where it is. The step
// bodies are still the old verbs, so what these check is the walking.

// reported is the step names the manager collected lines for, in order.
func reported(mg *Manager) []string {
	var out []string
	for _, line := range mg.Steps() {
		name, _, _ := strings.Cut(line, ":")
		out = append(out, name)
	}
	return out
}

// upRequest is the smallest request the stages that have moved will accept.
//
// It grows as stages move in and start reading more of it, which is why it is
// one function rather than a literal at each call: the alternative is editing
// every test in this file nine times.
// stubBinary is an executable that does nothing and succeeds.
//
// The init stage runs the node binary, and a unit test has no chain build. What
// it is checking is that the stage ran and reported, not what a real init
// writes; the live smoke of each commit is where a real binary is used.
func stubBinary(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stubchain")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func upRequest(t *testing.T) ChainUpIn {
	t.Helper()
	// Keys generated into this test's own directory, rather than the committed
	// preset: a test that walks the stages should not also depend on a ring
	// outside the package, nor write into one.
	return ChainUpIn{
		Chain: "stablenet", BPCount: 2, ENCount: 1,
		KeysSource: "generate", KeysDir: t.TempDir(),
		Binary: stubBinary(t),
	}
}

// withStage is upRequest with the stage it should stop at.
func withStage(in ChainUpIn, stage UpStage) ChainUpIn {
	in.Stage = stage
	return in
}

// newTestManager returns a manager over a fresh workspace and the walker
// standing in for the step bodies that have not moved yet.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	ws, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	return NewManager(Deps{}, ws)
}

// TestManagerTreeIsTheTreeTheDesignDrew.
//
// The tree is the design's picture, and a picture nobody compares drifts from
// the code under it. The leaves are one adapter per stage for now; each becomes
// a stage of its own with its own leaves, a commit at a time, and this is where
// that will show.
func TestManagerTreeIsTheTreeTheDesignDrew(t *testing.T) {
	mg := newTestManager(t)
	want := strings.Join([]string{
		"Composition",
		"  Stopped",
		"  Comparing",
		"    RestartingNodes",
		"    StoppingToRebuild",
		"  Composing",
		"    OpeningWorkspace",
		"    BuildingNodeTable",
		"    EnsuringKeys",
		"      KeysFromPreset",
		"      KeysGenerated",
		"      KeysDeclared",
		"    Reconciling",
		"    BuildingGenesis",
		"      GenesisFromTemplate",
		"      GenesisFromExisting",
		"    BuildingNodeConfig",
		"    BuildingNodeCommand",
		"    DeployingInputs",
		"      InputsVerifiedLocal",
		"      InputsShippedRemote",
		"    InitializingDatadirs",
		"    Launching",
		"      LaunchingPhase",
		"      RunningPhaseActions",
		"      RecordingRun",
		"  Composed",
		"  Verifying",
		"  Ready",
		"    Stopping",
		"    Restarting",
		"    Swapping",
		"    Hardforking",
		"    CrossingFork",
		"    Removing",
		"  Failed",
		"",
	}, "\n")
	if got := mg.Tree(); got != want {
		t.Errorf("the tree is\n%s\nwant\n%s", got, want)
	}
}

// TestStageOrderIsUpStepNames: the order the machine walks and the order a
// record is read in have to be one order, or a resume lands on a different
// stage than the one that failed.
func TestStageOrderIsUpStepNames(t *testing.T) {
	var steps []string
	for _, s := range stageOrder {
		steps = append(steps, s.step)
	}
	if !slices.Equal(steps, UpStepNames) {
		t.Errorf("the machine walks %v, the record is read as %v", steps, UpStepNames)
	}
}

// TestCompose_WalksEveryStageInOrderAndEndsComposed.
//
// It stops at the deploy stage. Every stage does its own work now, so a walk
// past it would init datadirs and launch a network, which is what the live run
// at the end of each commit is for; what this holds is the order.
func TestCompose_WalksEveryStageInOrderAndEndsComposed(t *testing.T) {
	mg := newTestManager(t)
	if err := mg.Compose(context.Background(), withStage(upRequest(t), UpDeploy), ""); err != nil {
		t.Fatal(err)
	}
	want := UpStepNames[:7]
	if got := reported(mg); !slices.Equal(got, want) {
		t.Errorf("reported %v, want %v", got, want)
	}
	if got := mg.m.Path(mg.m.Current()); got != "Composition/Composed" {
		t.Errorf("finished at %q, want Composition/Composed", got)
	}
}

// TestCompose_StopsWhereTheRequestSaid is --stage=deploy.
//
// Where a run stops used to be a target the caller computed and the loop
// compared against on every turn. Now it is one thing the stage parent knows,
// and the caller says nothing about it beyond the request it already had.
func TestCompose_StopsWhereTheRequestSaid(t *testing.T) {
	mg := newTestManager(t)
	if err := mg.Compose(context.Background(), withStage(upRequest(t), UpDeploy), ""); err != nil {
		t.Fatal(err)
	}
	if got := entered(mg); slices.Contains(got, "InitializingDatadirs") {
		t.Errorf("a run told to stop at deploy went on to init: %v", got)
	}
}

// TestCompose_BeginsAtTheNamedStep.
//
// Begun part way, on a workspace nothing has composed, so the named stage is
// the first thing entered and then fails for want of what comes before it.
// Failing is the point: it shows the walk really started there.
func TestCompose_BeginsAtTheNamedStep(t *testing.T) {
	mg := newTestManager(t)
	err := mg.Compose(context.Background(), upRequest(t), "config")
	if err == nil {
		t.Fatal("a composition begun at config on an empty workspace was accepted")
	}
	got := entered(mg)
	if len(got) == 0 || got[0] != "BuildingNodeConfig" {
		t.Errorf("the walk went through %v, and did not begin at BuildingNodeConfig", got)
	}
	if slices.Contains(got, "OpeningWorkspace") {
		t.Error("it ran a stage before the one it was told to begin at")
	}
}

// TestCompose_RefusesAStepItDoesNotHave, before it starts anything.
func TestCompose_RefusesAStepItDoesNotHave(t *testing.T) {
	mg := newTestManager(t)
	err := mg.Compose(context.Background(), upRequest(t), "nosuchstep")
	if err == nil {
		t.Fatal("an unknown step was accepted")
	}
	if !strings.Contains(err.Error(), "nosuchstep") {
		t.Errorf("refused with %q, want it to name the step", err)
	}
	if got := entered(mg); len(got) != 0 {
		t.Errorf("a refused request still entered %v", got)
	}
}

// TestCompose_AFailedStageStopsTheWalkAndKeepsTheReason.
func TestCompose_AFailedStageStopsTheWalkAndKeepsTheReason(t *testing.T) {
	mg := newTestManager(t)
	in := upRequest(t)
	// A genesis named and not there: the stage that reads it refuses.
	in.GenesisExisting = filepath.Join(t.TempDir(), "nosuch.json")
	err := mg.Compose(context.Background(), in, "")
	if err == nil {
		t.Fatal("a genesis that is not there was accepted")
	}
	want := []string{"new", "place", "keys"}
	if got := reported(mg); !slices.Equal(got, want) {
		t.Errorf("reported %v, want %v — a stage after the failure ran", got, want)
	}
	if got := mg.m.Path(mg.m.Current()); got != "Composition/Failed" {
		t.Errorf("stopped at %q, want Composition/Failed", got)
	}
}

// TestFailed_RefusesEverythingButBeingCleared.
//
// A failure somebody has to leave on purpose is the difference between a
// composition that stopped and one that was composed over.
func TestFailed_RefusesEverythingButBeingCleared(t *testing.T) {
	mg := newTestManager(t)
	if err := mg.Compose(context.Background(), ChainUpIn{Chain: "nosuchchain"}, ""); err == nil {
		t.Fatal("a chain nothing registers was accepted")
	}
	if err := mg.m.Send(context.Background(), Compose{}); err == nil {
		t.Error("a failed composition accepted another request")
	}
	if err := mg.m.Send(context.Background(), ClearError{}); err != nil {
		t.Fatalf("the failure could not be cleared: %v", err)
	}
	if got := mg.m.Path(mg.m.Current()); got != "Composition/Stopped" {
		t.Errorf("a cleared failure left the machine at %q, want Composition/Stopped", got)
	}
}

// TestCompose_RecordsWhereItIsBeforeTheStageRuns is what the recorded path is
// for.
//
// A position written after a stage finishes can never name the stage that did
// not, which is the only question anybody asks a dead composition. So a
// composition is stopped inside a stage and the record is read where it fell.
func TestCompose_RecordsWhereItIsBeforeTheStageRuns(t *testing.T) {
	mg := newTestManager(t)
	in := upRequest(t)
	in.GenesisExisting = filepath.Join(t.TempDir(), "nosuch.json")
	if err := mg.Compose(context.Background(), in, ""); err == nil {
		t.Fatal("a genesis that is not there was accepted")
	}
	ws, err := Open(mg.ws.Dir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	// The stage that did not finish, not the word "failed": that is what a
	// resume re-enters, and the step record below is where the failure is.
	if got := ws.State().StatePath; got != "Composition/Composing/BuildingGenesis/GenesisFromExisting" {
		t.Errorf("the record says %q, want the stage that did not finish", got)
	}
	if step, ok := ws.State().Steps["genesis"]; !ok || step.Err == "" {
		t.Errorf("the record does not name genesis as the failed step: %+v", ws.State().Steps)
	}
}

// TestOpeningWorkspace_RecordsTheChainAndTheRequestTogether is the first stage
// that does its own work.
//
// The two writes were two opens and two saves before: the verb recorded the
// chain, then a second pass recorded the request. A run that died between them
// left a workspace naming a chain it had no request for, which is a composition
// a resume cannot continue. One open, one save, both or neither.
func TestOpeningWorkspace_RecordsTheChainAndTheRequestTogether(t *testing.T) {
	mg := newTestManager(t)
	in := upRequest(t)
	in.BPCount = 3
	if err := mg.Compose(context.Background(), in, ""); err != nil {
		t.Fatal(err)
	}
	ws, err := Open(mg.ws.Dir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	st := ws.State()
	if st.Chain != "stablenet" {
		t.Errorf("the record names chain %q, want stablenet", st.Chain)
	}
	if st.Request == nil || st.Request.BPCount != 3 {
		t.Errorf("the record kept request %+v, want the one that was composed", st.Request)
	}
	if !strings.HasPrefix(mg.Steps()[0], "new: ") {
		t.Errorf("the stage reported %q, want a line beginning \"new: \"", mg.Steps()[0])
	}
}

// TestOpeningWorkspace_AnUnknownChainFailsTheComposition, and the record says
// which stage it was.
func TestOpeningWorkspace_AnUnknownChainFailsTheComposition(t *testing.T) {
	mg := newTestManager(t)
	err := mg.Compose(context.Background(), ChainUpIn{Chain: "nosuchchain"}, "")
	if err == nil {
		t.Fatal("a chain nothing registers was accepted")
	}
	if !strings.Contains(err.Error(), "new") {
		t.Errorf("refused with %q, want it to name the stage", err)
	}
	if got := entered(mg); slices.Contains(got, "BuildingNodeTable") {
		t.Errorf("a stage after the failure ran: %v", got)
	}
	ws, oerr := Open(mg.ws.Dir(), nil)
	if oerr != nil {
		t.Fatal(oerr)
	}
	if got := ws.State().StatePath; got != "Composition/Composing/OpeningWorkspace" {
		t.Errorf("the record says %q, want the stage that did not finish", got)
	}
	if step, ok := ws.State().Steps["new"]; !ok || step.Err == "" {
		t.Errorf("the record does not name new as the failed step: %+v", ws.State().Steps)
	}
}

// TestBuildingNodeTable_PlacesTheNodesTheRequestAsksFor.
func TestBuildingNodeTable_PlacesTheNodesTheRequestAsksFor(t *testing.T) {
	mg := newTestManager(t)
	in := upRequest(t)
	in.BPCount, in.ENCount = 3, 2
	if err := mg.Compose(context.Background(), in, ""); err != nil {
		t.Fatal(err)
	}
	ws, err := Open(mg.ws.Dir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(ws.State().Nodes); got != 5 {
		t.Errorf("the record holds %d nodes, want 5", got)
	}
	if got := ws.State().BPCount; got != 3 {
		t.Errorf("the record says %d producers, want 3", got)
	}
	if !strings.HasPrefix(mg.Steps()[1], "place: ") {
		t.Errorf("the stage reported %q, want a line beginning \"place: \"", mg.Steps()[1])
	}
}

// TestBuildingNodeTable_RefusesTwoLayouts, and says which stage refused.
//
// A blueprint and a topology are two descriptions of the same thing, and
// preferring either silently composes a network the reader did not ask for.
func TestBuildingNodeTable_RefusesTwoLayouts(t *testing.T) {
	mg := newTestManager(t)
	in := upRequest(t)
	in.TopologyPath = filepath.Join(t.TempDir(), "topology.yaml")
	in.BlueprintPath = filepath.Join(t.TempDir(), "blueprint.json")
	if err := os.WriteFile(in.BlueprintPath, []byte(`{"schemaVersion":"1","chain":"stablenet"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	err := mg.Compose(context.Background(), in, "")
	if err == nil {
		t.Fatal("two layouts were accepted")
	}
	if !strings.Contains(err.Error(), "place") {
		t.Errorf("refused with %q, want it to name the stage", err)
	}
	ws, oerr := Open(mg.ws.Dir(), nil)
	if oerr != nil {
		t.Fatal(oerr)
	}
	if got := ws.State().StatePath; got != "Composition/Composing/BuildingNodeTable" {
		t.Errorf("the record says %q, want the stage that did not finish", got)
	}
}

// entered is the states the machine moved into, in order, from its own log.
func entered(mg *Manager) []string {
	var out []string
	for _, rec := range mg.m.Dump() {
		if rec.Dest != "" {
			out = append(out, string(rec.Dest))
		}
	}
	return out
}

// TestEnsuringKeys_SaysWhichWayTheKeysCameFrom is what this stage's leaves
// exist for.
//
// Two compositions of the same shape can end up with different validator
// addresses, and the answer is almost always that one took its identities from
// a preset and the other generated them. A stage that was one state could not
// tell them apart; a stage with three says it in the path it walks, which is
// what the record then keeps.
func TestEnsuringKeys_SaysWhichWayTheKeysCameFrom(t *testing.T) {
	cases := []struct {
		name string
		arm  func(in *ChainUpIn)
		want string
	}{
		{
			name: "generated into this run's own ring",
			arm:  func(*ChainUpIn) {},
			want: "KeysGenerated",
		},
		{
			name: "taken from the committed preset",
			arm: func(in *ChainUpIn) {
				in.KeysSource = ""
				in.KeysDir = filepath.Join("..", "..", "presets", "keys")
				// The preset ring holds five identities.
				in.BPCount, in.ENCount = 4, 1
			},
			want: "KeysFromPreset",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mg := newTestManager(t)
			in := upRequest(t)
			c.arm(&in)
			if err := mg.Compose(context.Background(), in, ""); err != nil {
				t.Fatal(err)
			}
			got := entered(mg)
			if !slices.Contains(got, c.want) {
				t.Errorf("the machine went through %v, and never entered %s", got, c.want)
			}
			for _, other := range []string{"KeysFromPreset", "KeysGenerated", "KeysDeclared"} {
				if other != c.want && slices.Contains(got, other) {
					t.Errorf("it also entered %s, and a composition takes one way", other)
				}
			}
		})
	}
}

// TestBuildingGenesis_SaysWhereTheGenesisCameFrom.
//
// A genesis built from the family's template is one this run decided; a genesis
// taken from a file is one somebody else decided and this run is bound to. A
// chain that will not start looks the same either way until you know which.
func TestBuildingGenesis_SaysWhereTheGenesisCameFrom(t *testing.T) {
	// A genesis this run wrote, to hand to the second case as an existing one.
	first := newTestManager(t)
	firstIn := upRequest(t)
	if err := first.Compose(context.Background(), firstIn, ""); err != nil {
		t.Fatal(err)
	}
	if got := entered(first); !slices.Contains(got, "GenesisFromTemplate") {
		t.Fatalf("a run with no genesis named went through %v, and never built one from the template", got)
	}
	existing := filepath.Join(first.ws.Dir(), "genesis.json")

	second := newTestManager(t)
	in := upRequest(t)
	// The same ring as the first run: a genesis names its validators, and one
	// composed against different identities is refused before this test could
	// say anything about which state it went through.
	in.KeysDir = firstIn.KeysDir
	in.GenesisExisting = existing
	if err := second.Compose(context.Background(), in, ""); err != nil {
		t.Fatal(err)
	}
	got := entered(second)
	if !slices.Contains(got, "GenesisFromExisting") {
		t.Errorf("a run given a genesis went through %v, and never took it", got)
	}
	if slices.Contains(got, "GenesisFromTemplate") {
		t.Error("it also built one from the template, and a composition takes one way")
	}
}

// TestDeployingInputs_SaysWhetherAnythingWasShipped.
//
// The branch here stands after the work: a deploy to a local target ships
// nothing, because the key set already is the place the config points at, and
// the only way to know is to count. So the state the composition lands in is
// the answer, and it is what the record keeps.
func TestDeployingInputs_SaysWhetherAnythingWasShipped(t *testing.T) {
	mg := newTestManager(t)
	if err := mg.Compose(context.Background(), upRequest(t), ""); err != nil {
		t.Fatal(err)
	}
	got := entered(mg)
	if !slices.Contains(got, "InputsVerifiedLocal") {
		t.Errorf("a local composition went through %v, and never verified its inputs in place", got)
	}
	if slices.Contains(got, "InputsShippedRemote") {
		t.Error("a local composition shipped something, and a local target has nowhere to ship to")
	}
	// The stage still reports its line, from the state that named the outcome.
	if !strings.HasPrefix(mg.Steps()[6], "deploy: ") {
		t.Errorf("the stage reported %q, want a line beginning \"deploy: \"", mg.Steps()[6])
	}
}

// TestResumeStep_ReadsThePositionTheMachineRecorded.
//
// This is what the recorded path was added for. A run that died in a stage
// names that stage, so a resume does that stage over rather than working the
// answer out a second way and hoping the two agree.
func TestResumeStep_ReadsThePositionTheMachineRecorded(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
	}{
		{"died in the genesis stage", "Composition/Composing/BuildingGenesis/GenesisFromTemplate", "genesis"},
		{"died in the launch, third phase", "Composition/Composing/Launching/LaunchingPhase", "start"},
		{"died on the way in", "Composition/Composing/OpeningWorkspace", "new"},
		{"finished", "Composition/Ready", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ws, err := Open(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			ws.SetStatePath(c.path)
			if got := ws.ResumeStep(); got != c.want {
				t.Errorf("a composition at %q resumes at %q, want %q", c.path, got, c.want)
			}
		})
	}
}

// TestResumeStep_AWorkspaceWithNoPositionFallsBackToTheStepMap.
//
// Composed and Stopped are deliberate stops rather than places a run is in, so
// what should follow is the request's to answer, not the position's.
func TestResumeStep_AWorkspaceWithNoPositionFallsBackToTheStepMap(t *testing.T) {
	mg := newTestManager(t)
	if err := mg.Compose(context.Background(), withStage(upRequest(t), UpDeploy), ""); err != nil {
		t.Fatal(err)
	}
	ws, err := Open(mg.ws.Dir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := ws.State().StatePath; got != "Composition/Composed" {
		t.Fatalf("the run stopped at %q, want Composition/Composed", got)
	}
	// A request that asked to stop at deploy has nothing left to resume.
	if got := ws.ResumeStep(); got != "" {
		t.Errorf("a composition told to stop at deploy resumes at %q", got)
	}
}

// TestComparing_EachVerdictGoesItsOwnWay holds the two vocabularies together.
//
// preflight answers how much has to be rebuilt; this says where that answer
// puts the run. A verdict added there without a move here would otherwise be
// found at a comparison that had already been made, against a live network.
func TestComparing_EachVerdictGoesItsOwnWay(t *testing.T) {
	mg := newTestManager(t)
	c := mg.comparing
	for _, tc := range []struct {
		v    preflight.Verdict
		want statemachine.StateName
	}{
		{preflight.Reuse, nameVerifying},
		{preflight.RebuildNodes, nameRestartingNodes},
		{preflight.RebuildAll, nameStoppingToRebuild},
		{preflight.Compose, nameOpeningWorkspace},
	} {
		got, err := c.nextFor(tc.v)
		if err != nil {
			t.Errorf("%s: %v", tc.v, err)
			continue
		}
		if got.Name() != tc.want {
			t.Errorf("%s goes to %s, want %s", tc.v, got.Name(), tc.want)
		}
	}
	// A verdict nobody declared is refused rather than guessed at.
	if _, err := c.nextFor(preflight.Verdict(9)); err == nil {
		t.Error("an undeclared verdict was given a move")
	}
}

// TestComposeComparing_NothingComposedComposes: the comparison against an empty
// target is "nothing is composed", and that walks into the stages.
func TestComposeComparing_NothingComposedComposes(t *testing.T) {
	mg := newTestManager(t)
	in := withStage(upRequest(t), UpDeploy)
	in.DataDir = mg.ws.Dir()
	if err := mg.ComposeComparing(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	got := entered(mg)
	if !slices.Contains(got, "Comparing") {
		t.Errorf("the run went through %v, and never compared", got)
	}
	if !slices.Contains(got, "OpeningWorkspace") {
		t.Errorf("the run went through %v, and never composed", got)
	}
	if mg.Decision() == "" {
		t.Error("the comparison reported nothing")
	}
}
