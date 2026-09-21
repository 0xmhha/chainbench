package chainsetup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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

func walk(t *testing.T, from string, stage UpStage, r *recorder) (*lifecycle.Machine, error) {
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
	if _, err := walk(t, "", UpStart, r); err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.ran, ",") != strings.Join(UpStepNames[:], ",") {
		t.Errorf("ran %v, want %v", r.ran, UpStepNames)
	}
}

// TestTheWalkStopsWhereTheStageSays is the comparison that used to sit inside
// the loop, asked once as a target instead.
func TestTheWalkStopsWhereTheStageSays(t *testing.T) {
	r := &recorder{}
	m, err := walk(t, "", UpDeploy, r)
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
	if _, err := walk(t, "config", UpStart, r); err != nil {
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
		if _, err := walk(t, "", UpStart, r); err != nil {
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

// TestFailuresBecomeTheirOwnStates is the other half of a stage moving in: the
// kinds a caller can branch on, and a default that says so rather than picking
// the nearest.
func TestFailuresBecomeTheirOwnStates(t *testing.T) {
	for _, c := range []struct {
		name string
		step string
		err  error
		// got is how far the step reached before failing.
		got  []lifecycle.Status
		want lifecycle.Status
	}{
		{"unknown key source", "keys", ofKind(errKeySourceUnknown, errors.New("x")), nil, lifecycle.ChainEnsureKeysFailUnknownSource},
		{"too few identities", "keys", ofKind(errKeyCountShort, errors.New("x")), nil, lifecycle.ChainEnsureKeysFailCountMismatch},
		{"a key that is not local", "keys", ofKind(errKeyRefNotLocal, errors.New("x")), nil, lifecycle.ChainEnsureKeysFailKeyNotLocal},
		{"an unreadable key", "keys", ofKind(errKeyUnreadable, errors.New("x")), nil, lifecycle.ChainEnsureKeysFailKeyUnreadable},
		{"something the key store refused", "keys", errors.New("no"), nil, lifecycle.FailStageUnclassified},

		{"a genesis that is not usable", "genesis", ofKind(errGenesisExistingInvalid, errors.New("x")), nil, lifecycle.ChainBuildGenesisFailExistingInvalid},
		{"a genesis of another network", "genesis", ofKind(errGenesisExistingForeign, errors.New("x")), nil, lifecycle.ChainBuildGenesisFailExistingForeign},
		// These two can only happen once a genesis exists, and the table says
		// so by listing them under the built states rather than the entry.
		{"a fork with nothing to resolve to", "genesis", ofKind(errGenesisForkUnresolved, errors.New("x")),
			[]lifecycle.Status{lifecycle.ChainBuildGenesisFromTemplate}, lifecycle.ChainBuildGenesisFailForkUnresolved},
		{"a declaration nobody runs", "genesis", ofKind(errGenesisDeclUnused, errors.New("x")),
			[]lifecycle.Status{lifecycle.ChainBuildGenesisFromTemplate}, lifecycle.ChainBuildGenesisFailDeclUnused},
		{"a target that cannot build it", "genesis", ofKind(errGenesisTargetUnable, errors.New("x")), nil, lifecycle.ChainBuildGenesisFailTargetUnable},
		{"something the genesis package refused", "genesis", errors.New("no"), nil, lifecycle.FailStageUnclassified},

		// A stage that has not moved in reports the debt state whatever failed.
		{"no binary", "start", ofKind(errLaunchNoBinary, errors.New("x")), nil, lifecycle.ChainLaunchNodesFailNoBinary},
		{"a busy port", "start", ofKind(errLaunchPortBusy, errors.New("x")), nil, lifecycle.ChainLaunchNodesFailPortBusy},
		{"the machine is occupied", "start", ofKind(errLaunchOccupied, errors.New("x")), nil, lifecycle.ChainLaunchNodesFailOccupied},
		// These two can only happen once a phase has begun, and the table lists
		// them under the phase states rather than the entry.
		{"a producer with no keystore", "start", ofKind(errLaunchNoKeystore, errors.New("x")),
			[]lifecycle.Status{lifecycle.ChainLaunchNodesPhaseLaunching}, lifecycle.ChainLaunchNodesFailNoKeystore},
		{"a phase with nowhere to act", "start", ofKind(errLaunchPhaseEmpty, errors.New("x")),
			[]lifecycle.Status{lifecycle.ChainLaunchNodesPhaseLaunching, lifecycle.ChainLaunchNodesPhaseActions},
			lifecycle.ChainLaunchNodesFailPhaseEmpty},
		{"something the driver refused", "start", errors.New("no"), nil, lifecycle.FailStageUnclassified},

		{"an input that is not there", "deploy", ofKind(errDeployInputMissing, errors.New("x")), nil, lifecycle.ChainDeployNodesFailInputMissing},
		{"an input somebody else wrote", "deploy", ofKind(errDeployInputForeign, errors.New("x")), nil, lifecycle.ChainDeployNodesFailInputForeign},

		{"no chain named", "new", ofKind(errNewNoChain, errors.New("x")), nil, lifecycle.ChainOpenWorkspaceFailNoChain},
		{"a bad launch option", "build", ofKind(errBuildBadOption, errors.New("x")), nil, lifecycle.ChainBuildNodeCommandFailBadOption},

		{"a layout given twice", "place", ofKind(errPlaceTwoLayouts, errors.New("x")), nil, lifecycle.ChainBuildNodeTableFailTwoLayouts},
		{"a contended server set", "place", ofKind(errPlaceSetContended, errors.New("x")), nil, lifecycle.ChainBuildNodeTableFailSetContended},
		{"a layout that cannot exist", "place", errors.New("no validator"), nil, lifecycle.FailStageUnclassified},

		{"a target that cannot initialize", "init", ofKind(errInitTargetUnable, errors.New("x")), nil, lifecycle.ChainInitNodesFailTargetUnable},
		{"a genesis that cannot be read back", "init", ofKind(errInitGenesisUnreadable, errors.New("x")), nil, lifecycle.ChainInitNodesFailGenesisUnreadable},
		{"a datadir that will not clear", "init", ofKind(errInitDatadir, errors.New("x")), nil, lifecycle.ChainInitNodesFailDatadir},
		// The init stage asks the launch's question before it writes anything,
		// and the answer keeps the launch's name.
		{"a busy port, noticed by init", "init", ofKind(errLaunchPortBusy, errors.New("x")), nil, lifecycle.ChainLaunchNodesFailPortBusy},
		{"the binary refusing the genesis", "init", errors.New("no"), nil, lifecycle.FailStageUnclassified},

		{"a bad override", "config", ofKind(errConfigBadOverride, errors.New("x")), nil, lifecycle.ChainBuildNodeConfigFailBadOverride},
		{"a config that did not read back", "config", ofKind(errConfigReadback, errors.New("x")), nil, lifecycle.ChainBuildNodeConfigFailReadback},
		{"a pinned input that cannot be read", "config", ofKind(errConfigPinUnreadable, errors.New("x")), nil, lifecycle.ChainBuildNodeConfigFailPinUnreadable},

		// A kind that belongs to another stage is not this stage's failure.
		{"a failure with no state here", "build", ofKind(errKeyRefNotLocal, errors.New("x")), nil, lifecycle.FailStageUnclassified},
	} {
		r := &recorder{failAt: c.step, fail: c.err, failPath: c.got}
		m, err := walk(t, "", UpStart, r)
		if err == nil {
			t.Fatalf("%s: the walk did not stop", c.name)
		}
		if m.At() != c.want {
			t.Errorf("%s: stopped at %s, want %s", c.name, m.At(), c.want)
		}
	}
}

// TestADebtStateSaysWhyItIsOne holds the rule that reaching the debt state is
// never silent.
//
// No stage owes a whole classification any more, so the first case drives the
// shape a stage added later starts in, and the second is the one that happens
// today: a stage that classifies and met a failure none of its states covers.
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
				classify: BuildFailure},
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

// TestAKindKeepsTheMessage pins why the kinds ride alongside the error instead
// of in front of it: the sentence an operator reads is the one the step wrote.
func TestAKindKeepsTheMessage(t *testing.T) {
	inner := errors.New(`chainsetup: keys: node2: key reference "k.txt" is not a readable file`)
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

// TestEveryStageIsPlaced holds the stage table to the composition and counts
// the two debts. Both numbers only go down, and lowering one is how a stage
// moving in gets said out loud.
func TestEveryStageIsPlaced(t *testing.T) {
	if len(composition) != len(UpStepNames) {
		t.Fatalf("the stage table has %d stages, the composition runs %d steps",
			len(composition), len(UpStepNames))
	}
	assuming, owing := 0, 0
	for i, s := range composition {
		if s.step != UpStepNames[i] {
			t.Errorf("stage %d is %s, the composition runs %s there", i, s.step, UpStepNames[i])
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
// own tests and by a live run.
func TestTheRealRefusalsCarryTheirKind(t *testing.T) {
	// A key named as something other than a local file.
	err := checkNodeKeyRef(2, "srv://build1/keys/node2.key")
	if err == nil {
		t.Fatal("a key on a server was accepted")
	}
	if got := KeysFailure(err); got != lifecycle.ChainEnsureKeysFailKeyNotLocal {
		t.Errorf("the real refusal classified as %s, want ChainEnsureKeysFailKeyNotLocal", got)
	}

	// A fork with no name. applyFork refuses before it reads any state, so a
	// zero workspace is enough to reach it.
	var w Workspace
	_, _, ferr := w.applyFork([]byte(`{}`), GenesisFork{})
	if ferr == nil {
		t.Fatal("a fork with no name was accepted")
	}
	if got := GenesisFailure(ferr); got != lifecycle.ChainBuildGenesisFailForkUnresolved {
		t.Errorf("the real refusal classified as %s, want ChainBuildGenesisFailForkUnresolved", got)
	}

	// A workspace opened without naming a chain.
	var w3 Workspace
	_, nerr := w3.New(NewOpts{})
	if nerr == nil {
		t.Fatal("a workspace with no chain was accepted")
	}
	if got := NewFailure(nerr); got != lifecycle.ChainOpenWorkspaceFailNoChain {
		t.Errorf("the real refusal classified as %s, want ChainOpenWorkspaceFailNoChain", got)
	}

	// A --set that is not one.
	_, operr := ParseOverrides([]string{"=novalue"})
	if operr == nil {
		t.Fatal("a --set with no key was accepted")
	}
	if got := BuildFailure(ofKind(errBuildBadOption, operr)); got != lifecycle.ChainBuildNodeCommandFailBadOption {
		t.Errorf("the real refusal classified as %s, want ChainBuildNodeCommandFailBadOption", got)
	}

	// A blueprint and a topology at once, refused before any workspace opens.
	// The blueprint has to be readable to get that far: a missing file is a
	// different refusal, which is what the first attempt at this test measured.
	bp := filepath.Join(t.TempDir(), "n.yaml")
	if werr := os.WriteFile(bp, []byte("# an empty blueprint is a valid one\n"), 0o600); werr != nil {
		t.Fatal(werr)
	}
	_, perr := NetAllocate(context.Background(), Deps{}, NetAllocateIn{
		DataDir: "/nonexistent", BlueprintPath: bp, TopologyPath: "t.yaml",
	})
	if perr == nil {
		t.Fatal("a layout described twice was accepted")
	}
	if got := PlaceFailure(perr); got != lifecycle.ChainBuildNodeTableFailTwoLayouts {
		t.Errorf("the real refusal classified as %s, want ChainBuildNodeTableFailTwoLayouts (%v)", got, perr)
	}

	// An override that is not key=value, refused where it is set.
	var w2 Workspace
	oerr := w2.recordConfigSet("all", []string{"nocolonhere"})
	if oerr == nil {
		t.Fatal("an override that is not key=value was accepted")
	}
	if got := ConfigFailure(oerr); got != lifecycle.ChainBuildNodeConfigFailBadOverride {
		t.Errorf("the real refusal classified as %s, want ChainBuildNodeConfigFailBadOverride", got)
	}

	// No binary at all, which checkBinary refuses before it touches a target.
	berr := checkBinary(context.Background(), nil, "")
	if berr == nil {
		t.Fatal("a launch with no binary was accepted")
	}
	if got := LaunchFailure(berr); got != lifecycle.ChainLaunchNodesFailNoBinary {
		t.Errorf("the real refusal classified as %s, want ChainLaunchNodesFailNoBinary", got)
	}
}
