package chainsetup

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// These check the walk itself, with the verbs replaced by a recorder. What they
// hold is the part the state machine took over from the loop: which steps run,
// in what order, how far, and where a run stops when one fails. Composing a
// real network is what the other tests in this package already do.

// recorder is a composeRun that writes down what it was asked to run, and can
// be told to fail one step.
type recorder struct {
	ran    []string
	failAt string
	fail   error
	// source is what the keys step reports having used. The real step reads it
	// off the source it chose; here it is set per case.
	source lifecycle.Status
}

func (r *recorder) run(step string) (lifecycle.Status, error) {
	r.ran = append(r.ran, step)
	if step == r.failAt {
		if r.fail != nil {
			return 0, r.fail
		}
		return 0, errors.New("the step said no")
	}
	if step == "keys" {
		if r.source == 0 {
			return lifecycle.ChainEnsureKeysFromPreset, nil
		}
		return r.source, nil
	}
	return 0, nil
}

func walk(t *testing.T, in NetUpIn, from string, stage UpStage, r *recorder) (*lifecycle.Machine, error) {
	t.Helper()
	start, err := startFor(from)
	if err != nil {
		t.Fatal(err)
	}
	target, err := targetFor(stage)
	if err != nil {
		t.Fatal(err)
	}
	m, err := lifecycle.New(start, target, upHandlers(in, r.run))
	if err != nil {
		t.Fatal(err)
	}
	return m, m.Run(context.Background())
}

func TestTheWalkRunsEveryStepInOrder(t *testing.T) {
	r := &recorder{}
	if _, err := walk(t, NetUpIn{}, "", UpStart, r); err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.ran, ",") != strings.Join(upStepNames[:], ",") {
		t.Errorf("ran %v, want %v", r.ran, upStepNames)
	}
}

// TestTheWalkStopsWhereTheStageSays is the comparison that used to sit inside
// the loop, asked once as a target instead.
func TestTheWalkStopsWhereTheStageSays(t *testing.T) {
	r := &recorder{}
	m, err := walk(t, NetUpIn{}, "", UpDeploy, r)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.ran, ","); got != "new,place,keys,genesis,config,build,deploy" {
		t.Errorf("ran %q", got)
	}
	if m.At() != lifecycle.ChainInitNodes {
		t.Errorf("stopped at %s, want ChainInitNodes", m.At())
	}
}

// TestTheWalkResumes starts where the record says the composition got to.
func TestTheWalkResumes(t *testing.T) {
	r := &recorder{}
	if _, err := walk(t, NetUpIn{}, "config", UpStart, r); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.ran, ","); got != "config,build,deploy,init,start" {
		t.Errorf("ran %q", got)
	}
}

// TestAFailedStepStopsAtItsDebtState pins what a failure looks like today: the
// reason is in the error and the state says which stage it came from and that
// nobody has classified it. Both halves matter — the state is what a reader
// sees first, and the debt list is what says why it is not more specific.
func TestAFailedStepStopsAtItsDebtState(t *testing.T) {
	r := &recorder{failAt: "genesis"}
	m, err := walk(t, NetUpIn{}, "", UpStart, r)
	if err == nil {
		t.Fatal("a step that failed did not stop the walk")
	}
	if !strings.Contains(err.Error(), "genesis") {
		t.Errorf("the error does not name the stage: %v", err)
	}
	if m.At() != lifecycle.FailStageUnclassified {
		t.Errorf("stopped at %s, want FailStageUnclassified", m.At())
	}
	if got := strings.Join(r.ran, ","); got != "new,place,keys,genesis" {
		t.Errorf("ran %q — a step ran after the failure", got)
	}
}

// TestEveryStageIsPlacedAndOwed holds the three tables together: every step the
// composition runs has a stage, and every stage still says why its failures are
// one error string. The second list only shrinks, so an entry leaving is a
// stage whose work has moved into its handler.
func TestEveryStageIsPlacedAndOwed(t *testing.T) {
	for _, name := range upStepNames {
		if _, ok := stageOf[name]; !ok {
			t.Errorf("step %s has no lifecycle stage", name)
		}
		_, owed := unclassified[name]
		_, classifies := failureOf[name]
		if owed == classifies {
			t.Errorf("step %s is in both lists or in neither — a stage either "+
				"classifies its failures or says why it cannot", name)
		}
	}
	if len(stageOf) != len(upStepNames) {
		t.Errorf("stageOf names %d steps, the composition runs %d", len(stageOf), len(upStepNames))
	}
	// This number only goes down. A stage leaves when its failures become
	// kinds, and lowering it is how that move gets said out loud.
	const stillOwed = 8
	if len(unclassified) != stillOwed {
		t.Errorf("the debt list names %d steps, the count says %d — "+
			"lower the count when a stage leaves, never raise it",
			len(unclassified), stillOwed)
	}
}

// TestTheKeysStageTakesTheSourceTheStepReports is the point of the keys stage
// moving in: the state follows what the step did, not what the request said.
func TestTheKeysStageTakesTheSourceTheStepReports(t *testing.T) {
	for _, want := range []lifecycle.Status{
		lifecycle.ChainEnsureKeysFromPreset,
		lifecycle.ChainEnsureKeysGenerated,
		lifecycle.ChainEnsureKeysFromBlueprint,
	} {
		// The request says nothing at all, which is the case the old inference
		// read as the preset every time.
		r := &recorder{source: want}
		if _, err := walk(t, NetUpIn{}, "", UpStart, r); err != nil {
			t.Fatalf("%s: %v", want, err)
		}
	}
}

// TestTheKeysStageRefusesAStepThatSaysNothing holds the other half: a step that
// reports no source is a step whose report was lost, and guessing one back is
// what this stage stopped doing.
func TestTheKeysStageRefusesAStepThatSaysNothing(t *testing.T) {
	h := upHandlers(NetUpIn{}, func(string) (lifecycle.Status, error) { return 0, nil })
	m, err := lifecycle.New(lifecycle.ChainEnsureKeys, lifecycle.ChainVerify, h)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Run(context.Background()); err == nil {
		t.Fatal("a keys step that named no source was taken as one")
	}
}

// TestKeysFailuresBecomeTheirOwnStates is the second half of the stage moving
// in: four kinds a caller can branch on, and a default that says so rather than
// picking the nearest.
func TestKeysFailuresBecomeTheirOwnStates(t *testing.T) {
	for _, c := range []struct {
		name string
		err  error
		want lifecycle.Status
	}{
		{"unknown source", ofKind(errKeySourceUnknown, errors.New("x")), lifecycle.ChainEnsureKeysFailUnknownSource},
		{"count short", ofKind(errKeyCountShort, errors.New("x")), lifecycle.ChainEnsureKeysFailCountMismatch},
		{"not local", ofKind(errKeyRefNotLocal, errors.New("x")), lifecycle.ChainEnsureKeysFailKeyNotLocal},
		{"unreadable", ofKind(errKeyUnreadable, errors.New("x")), lifecycle.ChainEnsureKeysFailKeyUnreadable},
		{"something else", errors.New("the key store said no"), lifecycle.FailStageUnclassified},
	} {
		if got := keysFailure(c.err); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
		r := &recorder{failAt: "keys", fail: c.err}
		m, err := walk(t, NetUpIn{}, "", UpStart, r)
		if err == nil {
			t.Fatalf("%s: the walk did not stop", c.name)
		}
		if m.At() != c.want {
			t.Errorf("%s: stopped at %s, want %s", c.name, m.At(), c.want)
		}
	}
}

// TestAKindKeepsTheMessage pins why the kinds ride alongside the error instead
// of in front of it: the sentence an operator reads is the one the step wrote.
func TestAKindKeepsTheMessage(t *testing.T) {
	inner := errors.New("chainsetup: keys: node2: key reference \"k.txt\" is not a readable file")
	marked := ofKind(errKeyRefNotLocal, inner)
	if marked.Error() != inner.Error() {
		t.Errorf("the message changed: %q", marked.Error())
	}
	if !errors.Is(marked, errKeyRefNotLocal) || !errors.Is(marked, inner) {
		t.Error("a marked error lost either its kind or its cause")
	}
	if ofKind(errKeyRefNotLocal, nil) != nil {
		t.Error("a nil error came back marked")
	}
}

// TestGenesisWalksWhatTheRequestAsksFor covers the pair of refinements: a fork
// and a genesis per binary are things asked for, not routes through the stage.
func TestGenesisWalksWhatTheRequestAsksFor(t *testing.T) {
	for _, c := range []struct {
		name string
		in   NetUpIn
	}{
		{"plain", NetUpIn{}},
		{"existing", NetUpIn{GenesisExisting: "genesis.json"}},
		{"forked", NetUpIn{GenesisFork: &GenesisFork{}}},
		{"per binary", NetUpIn{GenesisPerBinary: map[string]string{"old": "o.yaml"}}},
		{"both", NetUpIn{GenesisFork: &GenesisFork{}, GenesisPerBinary: map[string]string{"old": "o.yaml"}}},
	} {
		r := &recorder{}
		if _, err := walk(t, c.in, "", UpStart, r); err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if len(r.ran) != len(upStepNames) {
			t.Errorf("%s: ran %v", c.name, r.ran)
		}
	}
}
