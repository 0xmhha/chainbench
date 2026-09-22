package chainsetup

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
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

func (w *walker) run(_ context.Context, step string) error {
	if w.before != nil {
		w.before(step)
	}
	w.ran = append(w.ran, step)
	if step == w.failAt {
		return w.failErr
	}
	return nil
}

// newTestManager returns a manager over a fresh workspace and the walker
// standing in for the step bodies.
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
		"    BuildingGenesis",
		"    BuildingNodeConfig",
		"    BuildingNodeCommand",
		"    DeployingInputs",
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
func TestCompose_WalksEveryStageInOrderAndEndsReady(t *testing.T) {
	mg, w := newTestManager(t)
	if err := mg.Compose(context.Background(), NetUpIn{}, ""); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(w.ran, UpStepNames) {
		t.Errorf("ran %v, want %v", w.ran, UpStepNames)
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
	if err := mg.Compose(context.Background(), NetUpIn{Stage: UpDeploy}, ""); err != nil {
		t.Fatal(err)
	}
	want := []string{"new", "place", "keys", "genesis", "config", "build", "deploy"}
	if !slices.Equal(w.ran, want) {
		t.Errorf("ran %v, want %v", w.ran, want)
	}
	if got := mg.m.Path(mg.m.Current()); got != "Composition/Composed" {
		t.Errorf("stopped at %q, want Composition/Composed", got)
	}
}

// TestCompose_BeginsAtTheNamedStep is what a resume does today.
func TestCompose_BeginsAtTheNamedStep(t *testing.T) {
	mg, w := newTestManager(t)
	if err := mg.Compose(context.Background(), NetUpIn{}, "config"); err != nil {
		t.Fatal(err)
	}
	want := []string{"config", "build", "deploy", "init", "start"}
	if !slices.Equal(w.ran, want) {
		t.Errorf("ran %v, want %v", w.ran, want)
	}
}

// TestCompose_RefusesAStepItDoesNotHave, before it starts anything.
func TestCompose_RefusesAStepItDoesNotHave(t *testing.T) {
	mg, w := newTestManager(t)
	err := mg.Compose(context.Background(), NetUpIn{}, "nosuchstep")
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
	w.failAt = "genesis"
	err := mg.Compose(context.Background(), NetUpIn{}, "")
	if !errors.Is(err, w.failErr) {
		t.Fatalf("Compose returned %v, want the stage's own error", err)
	}
	want := []string{"new", "place", "keys", "genesis"}
	if !slices.Equal(w.ran, want) {
		t.Errorf("ran %v, want %v — a stage after the failure ran", w.ran, want)
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
	w.failAt = "keys"
	if err := mg.Compose(context.Background(), NetUpIn{}, ""); err == nil {
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
	if err := mg.Compose(context.Background(), NetUpIn{}, ""); err != nil {
		t.Fatal(err)
	}
	for _, s := range stageOrder {
		want := "Composition/Composing/" + string(s.name)
		if seen[s.step] != want {
			t.Errorf("during %s the record said %q, want %q", s.step, seen[s.step], want)
		}
	}
}

// TestCompose_TheRecordKeepsWhereItDied.
func TestCompose_TheRecordKeepsWhereItDied(t *testing.T) {
	mg, w := newTestManager(t)
	w.failAt = "deploy"
	if err := mg.Compose(context.Background(), NetUpIn{}, ""); err == nil {
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
