package chainsetup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // the opening stage checks the chain is one we have
)

// The machine's job at this commit is to walk the same nine steps in the same
// order the table walked, stop where it was told, and say where it is. The step
// bodies are still the old verbs, so what these check is the walking.

// walker is a step runner that writes down what it was asked to do, and can be
// told to fail at one step.
type walker struct {
	ran     []string
	failAt  string
	failErr error
	before  func(step string) // called before the step is marked as run
}

func (w *walker) run(_ context.Context, step string) (string, error) {
	if w.before != nil {
		w.before(step)
	}
	w.ran = append(w.ran, step)
	if step == w.failAt {
		return "", w.failErr
	}
	return step + " done", nil
}

// adapterSteps is the steps whose bodies have not moved into a state of their
// own yet, in order.
//
// Derived rather than written down: it shrinks by one at each commit of this
// series, and a list kept by hand here would be nine edits and nine chances to
// say the wrong thing.
func adapterSteps(mg *Manager) []string {
	var out []string
	for _, st := range mg.stages {
		if _, ok := st.(*legacyStage); ok {
			out = append(out, st.step())
		}
	}
	return out
}

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
func upRequest(t *testing.T) ChainUpIn {
	t.Helper()
	// Keys generated into this test's own directory, rather than the committed
	// preset: a test that walks the stages should not also depend on a ring
	// outside the package, nor write into one.
	return ChainUpIn{
		Chain: "stablenet", BPCount: 2, ENCount: 1,
		KeysSource: "generate", KeysDir: t.TempDir(),
	}
}

// withStage is upRequest with the stage it should stop at.
func withStage(in ChainUpIn, stage UpStage) ChainUpIn {
	in.Stage = stage
	return in
}

// newTestManager returns a manager over a fresh workspace and the walker
// standing in for the step bodies that have not moved yet.
func newTestManager(t *testing.T) (*Manager, *walker) {
	t.Helper()
	ws, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	w := &walker{failErr: errors.New("the stage refused")}
	return NewManager(Deps{}, ws, w.run), w
}

// TestManagerTreeIsTheTreeTheDesignDrew.
//
// The tree is the design's picture, and a picture nobody compares drifts from
// the code under it. The leaves are one adapter per stage for now; each becomes
// a stage of its own with its own leaves, a commit at a time, and this is where
// that will show.
func TestManagerTreeIsTheTreeTheDesignDrew(t *testing.T) {
	mg, _ := newTestManager(t)
	want := strings.Join([]string{
		"Composition",
		"  Stopped",
		"  Composing",
		"    OpeningWorkspace",
		"    BuildingNodeTable",
		"    EnsuringKeys",
		"      KeysFromPreset",
		"      KeysGenerated",
		"      KeysDeclared",
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
		"  Composed",
		"  Ready",
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

// TestCompose_WalksEveryStageInOrderAndEndsReady.
//
// The order is read off what the manager reported, not off what the walker was
// asked to do: a stage that has moved into its own state does not go through
// the walker, and the claim here is about all nine.
func TestCompose_WalksEveryStageInOrderAndEndsReady(t *testing.T) {
	mg, w := newTestManager(t)
	if err := mg.Compose(context.Background(), upRequest(t), ""); err != nil {
		t.Fatal(err)
	}
	if got := reported(mg); !slices.Equal(got, UpStepNames) {
		t.Errorf("reported %v, want %v", got, UpStepNames)
	}
	if want := adapterSteps(mg); !slices.Equal(w.ran, want) {
		t.Errorf("the walker ran %v, want the stages that have not moved yet, %v", w.ran, want)
	}
	if got := mg.m.Path(mg.m.Current()); got != "Composition/Ready" {
		t.Errorf("finished at %q, want Composition/Ready", got)
	}
}

// TestCompose_StopsWhereTheRequestSaid is --stage=deploy.
//
// Where a run stops used to be a target the caller computed and the loop
// compared against on every turn. Now it is one thing the stage parent knows,
// and the caller says nothing about it beyond the request it already had.
func TestCompose_StopsWhereTheRequestSaid(t *testing.T) {
	mg, w := newTestManager(t)
	if err := mg.Compose(context.Background(), withStage(upRequest(t), UpDeploy), ""); err != nil {
		t.Fatal(err)
	}
	want := []string{"new", "place", "keys", "genesis", "config", "build", "deploy"}
	if got := reported(mg); !slices.Equal(got, want) {
		t.Errorf("reported %v, want %v", got, want)
	}
	_ = w
	if got := mg.m.Path(mg.m.Current()); got != "Composition/Composed" {
		t.Errorf("stopped at %q, want Composition/Composed", got)
	}
}

// TestCompose_BeginsAtTheNamedStep is what a resume does today.
func TestCompose_BeginsAtTheNamedStep(t *testing.T) {
	mg, w := newTestManager(t)
	// A step that is still an adapter, so this says what it means -- that the
	// walk begins where it was told -- without also needing the stages before
	// it to have run.
	if err := mg.Compose(context.Background(), upRequest(t), "init"); err != nil {
		t.Fatal(err)
	}
	want := []string{"init", "start"}
	if got := reported(mg); !slices.Equal(got, want) {
		t.Errorf("reported %v, want %v", got, want)
	}
	_ = w
}

// TestCompose_RefusesAStepItDoesNotHave, before it starts anything.
func TestCompose_RefusesAStepItDoesNotHave(t *testing.T) {
	mg, w := newTestManager(t)
	err := mg.Compose(context.Background(), upRequest(t), "nosuchstep")
	if err == nil {
		t.Fatal("an unknown step was accepted")
	}
	if !strings.Contains(err.Error(), "nosuchstep") {
		t.Errorf("refused with %q, want it to name the step", err)
	}
	if len(w.ran) != 0 {
		t.Errorf("a refused request still ran %v", w.ran)
	}
}

// TestCompose_AFailedStageStopsTheWalkAndKeepsTheReason.
func TestCompose_AFailedStageStopsTheWalkAndKeepsTheReason(t *testing.T) {
	mg, w := newTestManager(t)
	w.failAt = "init"
	err := mg.Compose(context.Background(), upRequest(t), "")
	if !errors.Is(err, w.failErr) {
		t.Fatalf("Compose returned %v, want the stage's own error", err)
	}
	// The failing stage reports nothing, so the lines stop one short of it.
	want := []string{"new", "place", "keys", "genesis", "config", "build", "deploy"}
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
	mg, w := newTestManager(t)
	w.failAt = "init"
	if err := mg.Compose(context.Background(), upRequest(t), ""); err == nil {
		t.Fatal("the failing stage did not fail the composition")
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
// not, which is the only question anybody asks a dead composition. So the
// walker reads the record from inside each step and gets the step it is in.
func TestCompose_RecordsWhereItIsBeforeTheStageRuns(t *testing.T) {
	mg, w := newTestManager(t)
	dir := mg.ws.Dir()
	seen := map[string]string{}
	w.before = func(step string) {
		ws, err := Open(dir, nil)
		if err != nil {
			t.Errorf("reading the record during %s: %v", step, err)
			return
		}
		seen[step] = ws.State().StatePath
	}
	if err := mg.Compose(context.Background(), upRequest(t), ""); err != nil {
		t.Fatal(err)
	}
	for _, step := range adapterSteps(mg) {
		st, err := mg.stageFor(step)
		if err != nil {
			t.Fatal(err)
		}
		want := "Composition/Composing/" + string(st.Name())
		if seen[step] != want {
			t.Errorf("during %s the record said %q, want %q", step, seen[step], want)
		}
	}
}

// TestCompose_TheRecordKeepsWhereItDied.
func TestCompose_TheRecordKeepsWhereItDied(t *testing.T) {
	mg, w := newTestManager(t)
	w.failAt = "init"
	if err := mg.Compose(context.Background(), upRequest(t), ""); err == nil {
		t.Fatal("the failing stage did not fail the composition")
	}
	ws, err := Open(mg.ws.Dir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := ws.State().StatePath; got != "Composition/Failed" {
		t.Errorf("the record says %q, want Composition/Failed", got)
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
	mg, _ := newTestManager(t)
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
	mg, w := newTestManager(t)
	err := mg.Compose(context.Background(), ChainUpIn{Chain: "nosuchchain"}, "")
	if err == nil {
		t.Fatal("a chain nothing registers was accepted")
	}
	if !strings.Contains(err.Error(), "new") {
		t.Errorf("refused with %q, want it to name the stage", err)
	}
	if len(w.ran) != 0 {
		t.Errorf("a stage after the failure ran: %v", w.ran)
	}
	ws, oerr := Open(mg.ws.Dir(), nil)
	if oerr != nil {
		t.Fatal(oerr)
	}
	if got := ws.State().StatePath; got != "Composition/Failed" {
		t.Errorf("the record says %q, want Composition/Failed", got)
	}
	if step, ok := ws.State().Steps["new"]; !ok || step.Err == "" {
		t.Errorf("the record does not name new as the failed step: %+v", ws.State().Steps)
	}
}

// TestBuildingNodeTable_PlacesTheNodesTheRequestAsksFor.
func TestBuildingNodeTable_PlacesTheNodesTheRequestAsksFor(t *testing.T) {
	mg, _ := newTestManager(t)
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
	mg, _ := newTestManager(t)
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
	if got := ws.State().StatePath; got != "Composition/Failed" {
		t.Errorf("the record says %q, want Composition/Failed", got)
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
			mg, _ := newTestManager(t)
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

// TestEnsuringKeys_TheLeafIsWhatTheRecordKeeps: the path a leaf writes is the
// one on disk while that leaf is doing its work.
func TestEnsuringKeys_TheLeafIsWhatTheRecordKeeps(t *testing.T) {
	mg, _ := newTestManager(t)
	// Fail the first stage that is still an adapter, and read the record from
	// inside it: a stage records its own position on the way in, so what is on
	// disk by then names that stage -- which is the guarantee, one stage on.
	var atKeys string
	mg.run = func(_ context.Context, step string) (string, error) {
		if step == "init" {
			ws, err := Open(mg.ws.Dir(), nil)
			if err != nil {
				return "", err
			}
			atKeys = ws.State().StatePath
			return "", errors.New("stop here")
		}
		return step + " done", nil
	}
	if err := mg.Compose(context.Background(), upRequest(t), ""); err == nil {
		t.Fatal("the walk was meant to stop at init")
	}

	if atKeys != "Composition/Composing/InitializingDatadirs" {
		t.Errorf("the record said %q when the init stage began", atKeys)
	}
	if got := entered(mg); !slices.Contains(got, "KeysGenerated") {
		t.Errorf("the machine went through %v, and never entered KeysGenerated", got)
	}
}

// TestBuildingGenesis_SaysWhereTheGenesisCameFrom.
//
// A genesis built from the family's template is one this run decided; a genesis
// taken from a file is one somebody else decided and this run is bound to. A
// chain that will not start looks the same either way until you know which.
func TestBuildingGenesis_SaysWhereTheGenesisCameFrom(t *testing.T) {
	// A genesis this run wrote, to hand to the second case as an existing one.
	first, _ := newTestManager(t)
	firstIn := upRequest(t)
	if err := first.Compose(context.Background(), firstIn, ""); err != nil {
		t.Fatal(err)
	}
	if got := entered(first); !slices.Contains(got, "GenesisFromTemplate") {
		t.Fatalf("a run with no genesis named went through %v, and never built one from the template", got)
	}
	existing := filepath.Join(first.ws.Dir(), "genesis.json")

	second, _ := newTestManager(t)
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
	mg, _ := newTestManager(t)
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
