package verb

import (
	"context"
	"errors"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// These check the walk itself, with the verbs replaced by a recorder. What they
// hold is the part the state machine took over from the loop: which steps run,
// in what order, how far, which states a stage goes through, and where a run
// stops when one fails. Composing a real network is what the other tests in
// this package already do.

// recorder is a composeRun that writes down what it was asked to run. It can be
// told to fail one step, and to report a path for the steps that report one.
type recorder struct {
	ran    []string
	failAt string
	fail   error
	// failPath is how far the failing step got. A step that fails after it has
	// built something reports that, which is what decides the state its failure
	// is reached from.
	failPath []lifecycle.Status
	// path is what each step reports having gone through. A step with no entry
	// reports nothing, which is what a step whose work has not moved in does.
	path map[string][]lifecycle.Status
}

func (r *recorder) run(step string) ([]lifecycle.Status, error) {
	r.ran = append(r.ran, step)
	if step == r.failAt {
		if r.fail != nil {
			return r.failPath, r.fail
		}
		return r.failPath, errors.New("the step said no")
	}
	if p, ok := r.path[step]; ok {
		return p, nil
	}
	// The stages whose work has moved in must say something, so a recorder that
	// does not care gives them their plainest answer: one source, one genesis
	// built from the template, one launch phase.
	switch step {
	case "keys":
		return []lifecycle.Status{lifecycle.ChainEnsureKeysFromPreset}, nil
	case "genesis":
		return []lifecycle.Status{lifecycle.ChainBuildGenesisFromTemplate}, nil
	case "start":
		return []lifecycle.Status{
			lifecycle.ChainLaunchNodesPhaseLaunching,
			lifecycle.ChainLaunchNodesPhaseDone,
		}, nil
	case "deploy":
		return []lifecycle.Status{lifecycle.ChainDeployNodesVerifiedLocal}, nil
	}
	return nil, nil
}

func walk(t *testing.T, from string, stage chainsetup.UpStage, r *recorder) (*lifecycle.Machine, error) {
	t.Helper()
	start, err := startFor(from)
	if err != nil {
		t.Fatal(err)
	}
	target, err := targetFor(stage)
	if err != nil {
		t.Fatal(err)
	}
	m, err := lifecycle.New(start, target, upHandlers(r.run))
	if err != nil {
		t.Fatal(err)
	}
	return m, m.Run(context.Background())
}

func TestTheWalkRunsEveryStepInOrder(t *testing.T) {
	r := &recorder{}
	if _, err := walk(t, "", chainsetup.UpStart, r); err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.ran, ",") != strings.Join(chainsetup.UpStepNames[:], ",") {
		t.Errorf("ran %v, want %v", r.ran, chainsetup.UpStepNames)
	}
}

// TestTheWalkStopsWhereTheStageSays is the comparison that used to sit inside
// the loop, asked once as a target instead.
func TestTheWalkStopsWhereTheStageSays(t *testing.T) {
	r := &recorder{}
	m, err := walk(t, "", chainsetup.UpDeploy, r)
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
	if _, err := walk(t, "config", chainsetup.UpStart, r); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(r.ran, ","); got != "config,build,deploy,init,start" {
		t.Errorf("ran %q", got)
	}
}

// TestAStageWalksThePathItsStepReports is the point of the two stages that have
// moved in: the states follow what the step did, not what the request said.
//
// Every path here is one the transition table has to allow, so a path the step
// could report and the table could not take fails this rather than a run.
func TestAStageWalksThePathItsStepReports(t *testing.T) {
	for _, c := range []struct {
		name string
		path map[string][]lifecycle.Status
	}{
		{"keys from the preset", map[string][]lifecycle.Status{
			"keys": {lifecycle.ChainEnsureKeysFromPreset}}},
		{"keys generated", map[string][]lifecycle.Status{
			"keys": {lifecycle.ChainEnsureKeysGenerated}}},
		{"keys from a declaration", map[string][]lifecycle.Status{
			"keys": {lifecycle.ChainEnsureKeysFromBlueprint}}},
		{"a genesis taken verbatim", map[string][]lifecycle.Status{
			"genesis": {lifecycle.ChainBuildGenesisFromExisting}}},
		{"a fork scheduled on a built genesis", map[string][]lifecycle.Status{
			"genesis": {lifecycle.ChainBuildGenesisFromTemplate, lifecycle.ChainBuildGenesisForkApplied}}},
		{"a genesis per binary", map[string][]lifecycle.Status{
			"genesis": {lifecycle.ChainBuildGenesisFromTemplate, lifecycle.ChainBuildGenesisVariantsWritten}}},
		{"both, on a genesis taken verbatim", map[string][]lifecycle.Status{
			"genesis": {
				lifecycle.ChainBuildGenesisFromExisting,
				lifecycle.ChainBuildGenesisForkApplied,
				lifecycle.ChainBuildGenesisVariantsWritten}}},
		// A wbft network declares one phase; a poa network declares a boot and
		// one join per producer, and the boot phase carries actions.
		{"a deploy that shipped to a remote target", map[string][]lifecycle.Status{
			"deploy": {lifecycle.ChainDeployNodesShippedRemote}}},
		{"a launch of one phase", map[string][]lifecycle.Status{
			"start": {
				lifecycle.ChainLaunchNodesPhaseLaunching,
				lifecycle.ChainLaunchNodesPhaseDone}}},
		{"a launch whose first phase has actions", map[string][]lifecycle.Status{
			"start": {
				lifecycle.ChainLaunchNodesPhaseLaunching,
				lifecycle.ChainLaunchNodesPhaseActions,
				lifecycle.ChainLaunchNodesPhaseDone}}},
		{"a poa launch: boot with actions, then four joins", map[string][]lifecycle.Status{
			"start": {
				lifecycle.ChainLaunchNodesPhaseLaunching,
				lifecycle.ChainLaunchNodesPhaseActions,
				lifecycle.ChainLaunchNodesPhaseDone,
				lifecycle.ChainLaunchNodesPhaseLaunching, lifecycle.ChainLaunchNodesPhaseDone,
				lifecycle.ChainLaunchNodesPhaseLaunching, lifecycle.ChainLaunchNodesPhaseDone,
				lifecycle.ChainLaunchNodesPhaseLaunching, lifecycle.ChainLaunchNodesPhaseDone,
				lifecycle.ChainLaunchNodesPhaseLaunching, lifecycle.ChainLaunchNodesPhaseDone}}},
	} {
		r := &recorder{path: c.path}
		if _, err := walk(t, "", chainsetup.UpStart, r); err != nil {
			t.Errorf("%s: %v", c.name, err)
		}
	}
}

// TestAMovedStageRefusesAStepThatSaysNothing holds the other half. A stage
// whose work has moved in reports where it went; silence is a report that was
// lost, and filling one in is what these stages stopped doing.
func TestAMovedStageRefusesAStepThatSaysNothing(t *testing.T) {
	for _, step := range []string{"keys", "genesis", "start", "deploy"} {
		silent := func(s string) ([]lifecycle.Status, error) { return nil, nil }
		at := lifecycle.ChainEnsureKeys
		switch step {
		case "genesis":
			at = lifecycle.ChainBuildGenesis
		case "start":
			at = lifecycle.ChainLaunchNodes
		case "deploy":
			at = lifecycle.ChainDeployNodes
		}
		m, err := lifecycle.New(at, lifecycle.ChainVerify, upHandlers(silent))
		if err != nil {
			t.Fatal(err)
		}
		if err := m.Run(context.Background()); err == nil {
			t.Errorf("the %s stage took silence for an answer", step)
		}
	}
}

// TestADebtStateSaysWhyItIsOne holds the rule that reaching the debt state is
// never silent.
//
// Which state a failure is belongs to the stage that raises it, and is tested
// where those live. What is tested here is what the walk does with the answer:
// no stage owes a whole classification any more, so the first case drives the
// shape a stage added later starts in, and the second is the one that happens
// today — a stage that classifies and met a failure none of its states covers.
func TestADebtStateSaysWhyItIsOne(t *testing.T) {
	for _, c := range []struct {
		name  string
		stage composeStage
		says  string
	}{
		{"a stage with no classifier",
			composeStage{step: "example", at: lifecycle.ChainOpenWorkspace,
				owed: "the verb returns a string"},
			"example: the verb returns a string"},
		{"a stage whose classifier has no state for this",
			composeStage{step: "build", at: lifecycle.ChainBuildNodeCommand,
				classify: chainsetup.BuildFailure},
			"build: this failure has no state of its own"},
	} {
		m, err := lifecycle.New(c.stage.at, lifecycle.ChainVerify, upHandlers((&recorder{}).run))
		if err != nil {
			t.Fatal(err)
		}
		got := c.stage.failed(m, nil, errors.New("something went wrong"))
		if !strings.Contains(got.Error(), c.says) {
			t.Errorf("%s: %v", c.name, got)
		}
		if m.At() != lifecycle.FailStageUnclassified {
			t.Errorf("%s: at %s", c.name, m.At())
		}
	}
}

// TestEveryStageIsPlaced holds the stage table to the composition and counts
// the two debts. Both numbers only go down, and lowering one is how a stage
// moving in gets said out loud.
func TestEveryStageIsPlaced(t *testing.T) {
	if len(composition) != len(chainsetup.UpStepNames) {
		t.Fatalf("the stage table has %d stages, the composition runs %d steps",
			len(composition), len(chainsetup.UpStepNames))
	}
	assuming, owing := 0, 0
	for i, s := range composition {
		if s.step != chainsetup.UpStepNames[i] {
			t.Errorf("stage %d is %s, the composition runs %s there", i, s.step, chainsetup.UpStepNames[i])
		}
		if (s.classify == nil) != (s.owed != "") {
			t.Errorf("stage %s either classifies its failures or says why it cannot, not both or neither", s.step)
		}
		if len(s.assumed) > 0 {
			assuming++
		}
		if s.classify == nil {
			owing++
		}
	}
	if assuming != stagesStillAssuming {
		t.Errorf("%d stages assume a path, the count says %d — lower the count when a stage stops, never raise it",
			assuming, stagesStillAssuming)
	}
	if owing != stagesStillOwing {
		t.Errorf("%d stages owe a classification, the count says %d — lower the count when a stage leaves, never raise it",
			owing, stagesStillOwing)
	}
}

// TestTheRealRefusalsCarryTheirKind drives two refusal sites directly, because
// everything above this point hands the classifier an error built by hand.
//
// A kind that is attached in the test and not at the site is a classifier that
// passes its own tests and returns the debt state in production. These two are
// reachable without a composed workspace; the rest are covered by the stage's
