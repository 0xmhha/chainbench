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
}

func (r *recorder) run(step string) error {
	r.ran = append(r.ran, step)
	if step == r.failAt {
		return errors.New("the step said no")
	}
	return nil
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
		if _, ok := unclassified[name]; !ok {
			t.Errorf("step %s does not say why its failures are unclassified", name)
		}
	}
	if len(stageOf) != len(upStepNames) {
		t.Errorf("stageOf names %d steps, the composition runs %d", len(stageOf), len(upStepNames))
	}
	if len(unclassified) > len(upStepNames) {
		t.Errorf("the debt list names %d steps, more than the composition runs", len(unclassified))
	}
}

func TestKeysSourceStateReadsTheRequest(t *testing.T) {
	for _, c := range []struct {
		name string
		in   NetUpIn
		want lifecycle.Status
	}{
		{"silent", NetUpIn{}, lifecycle.ChainEnsureKeysFromPreset},
		{"named preset", NetUpIn{KeysSource: "keyPreset"}, lifecycle.ChainEnsureKeysFromPreset},
		{"generate", NetUpIn{KeysSource: "generate"}, lifecycle.ChainEnsureKeysGenerated},
		{"declared", NetUpIn{KeysSource: "declared"}, lifecycle.ChainEnsureKeysFromBlueprint},
		// The rule NetKeys applies: a declaration next to a silent source is
		// the source, so the operator does not name it twice.
		{"silent beside a blueprint", NetUpIn{BlueprintPath: "n.yaml"}, lifecycle.ChainEnsureKeysFromBlueprint},
		{"preset beside a blueprint", NetUpIn{KeysSource: "keyPreset", BlueprintPath: "n.yaml"}, lifecycle.ChainEnsureKeysFromPreset},
	} {
		if got := keysSourceState(c.in); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
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
