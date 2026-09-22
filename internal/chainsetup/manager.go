package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// StepRunner runs one composition step by name and returns the line it reports.
//
// The bodies live in the verb layer, which imports this package, so they are
// handed in rather than reached for. It is the same seam the composition
// already had: the runner closure the old machine's handlers were built from.
// It shrinks as stages move into their own states, and goes when the last one
// has.
type StepRunner func(ctx context.Context, step string) (detail string, err error)

// The composition's steps, as UpStepNames spells them.
const (
	stepNew     = "new"
	stepPlace   = "place"
	stepKeys    = "keys"
	stepGenesis = "genesis"
	stepConfig  = "config"
	stepBuild   = "build"
	stepDeploy  = "deploy"
	stepInit    = "init"
	stepStart   = "start"
)

// The states of a composition. The value is what a record keeps, through
// [statemachine.Machine.Path], so these strings are read by people and by the
// resume that will follow one.
const (
	nameComposition          statemachine.StateName = "Composition"
	nameStopped              statemachine.StateName = "Stopped"
	nameComposing            statemachine.StateName = "Composing"
	nameOpeningWorkspace     statemachine.StateName = "OpeningWorkspace"
	nameBuildingNodeTable    statemachine.StateName = "BuildingNodeTable"
	nameEnsuringKeys         statemachine.StateName = "EnsuringKeys"
	nameBuildingGenesis      statemachine.StateName = "BuildingGenesis"
	nameBuildingNodeConfig   statemachine.StateName = "BuildingNodeConfig"
	nameBuildingNodeCommand  statemachine.StateName = "BuildingNodeCommand"
	nameDeployingInputs      statemachine.StateName = "DeployingInputs"
	nameInitializingDatadirs statemachine.StateName = "InitializingDatadirs"
	nameLaunching            statemachine.StateName = "Launching"
	nameComposed             statemachine.StateName = "Composed"
	nameReady                statemachine.StateName = "Ready"
	nameFailed               statemachine.StateName = "Failed"
)

// stageOrder pairs each composition step with the state that runs it.
//
// The steps are UpStepNames and manager_test.go holds them to it, so the order
// a record is read in and the order the machine walks cannot drift apart.
var stageOrder = []struct {
	step string
	name statemachine.StateName
}{
	{stepNew, nameOpeningWorkspace},
	{stepPlace, nameBuildingNodeTable},
	{stepKeys, nameEnsuringKeys},
	{stepGenesis, nameBuildingGenesis},
	{stepConfig, nameBuildingNodeConfig},
	{stepBuild, nameBuildingNodeCommand},
	{stepDeploy, nameDeployingInputs},
	{stepInit, nameInitializingDatadirs},
	{stepStart, nameLaunching},
}

// Manager composes a network by walking states.
//
// It owns the machine and the states; it does not own the workspace's lock or
// the shape of what a caller prints. The workspace arrives open and held,
// because the lock covers more than the walk does — the reuse snapshot is taken
// before it and the node table is read after — and because a machine that took
// its own lock would make `up` and `run` lock differently, which is the
// difference the reconciliation commit exists to remove.
type Manager struct {
	d   Deps
	ws  *Workspace
	run StepRunner
	m   *statemachine.Machine

	// request is what this run was asked to compose. A stage reads the parts it
	// needs from it; it is stored once, when the request arrives.
	request ChainUpIn
	// stopAfter is the last step this run performs, or "" to run them all.
	stopAfter string
	// steps is one "step: detail" line per stage that finished, in order. It is
	// what a caller prints, and it is kept even when the run then failed,
	// because how far it got is the first thing a reader wants.
	steps []string
	// failure is why the composition stopped, written just before the move to
	// failed so that state can report it once.
	failure error

	stopped   *stoppedState
	composing *composingState
	stages    []stage
	composed  *composedState
	ready     *readyState
	failed    *failedState
}

// NewManager builds a composition machine over an open, held workspace.
//
// The Add calls below are the tree. They are written as the tree so the shape
// can be read rather than reconstructed, which is what the reference does with
// its own addState block.
func NewManager(d Deps, ws *Workspace, run StepRunner) *Manager {
	mg := &Manager{d: d, ws: ws, run: run, m: statemachine.New("composition", nil)}

	root := &compositionState{}
	mg.stopped = &stoppedState{mg: mg}
	mg.composing = &composingState{mg: mg}
	mg.composed = &composedState{mg: mg}
	mg.ready = &readyState{mg: mg}
	mg.failed = &failedState{mg: mg}
	// One state per stage. The ones that still say legacyStage run the old verb
	// through the injected runner; each commit of this series turns one of them
	// into a state that does the work itself.
	mg.stages = append(mg.stages, &openingWorkspace{mg: mg}, &buildingNodeTable{mg: mg},
		newEnsuringKeys(mg), newBuildingGenesis(mg), &buildingNodeConfig{mg: mg})
	for _, s := range stageOrder[len(mg.stages):] {
		mg.stages = append(mg.stages, &legacyStage{mg: mg, stepName: s.step, name: s.name})
	}

	mg.m.Add(root, nil)
	mg.m.Add(mg.stopped, root)
	mg.m.Add(mg.composing, root)
	for _, st := range mg.stages {
		mg.m.Add(st, mg.composing)
		// A stage with more than one way of doing its work says so, and its
		// ways go in right under it — the tree is read as a picture, and a
		// leaf three stages below its parent is not one.
		if branch, ok := st.(interface {
			leafStates() []statemachine.State
		}); ok {
			for _, leaf := range branch.leafStates() {
				mg.m.Add(leaf, st)
			}
		}
	}
	mg.m.Add(mg.composed, root)
	mg.m.Add(mg.ready, root)
	mg.m.Add(mg.failed, root)
	return mg
}

// Compose walks the composition described by the request.
//
// from names the step to begin at, for a resume; empty begins at the first.
// It is an argument rather than something read from the workspace because the
// record does not carry the machine's position yet — the field is there and
// nothing writes it until this commit, and nothing reads it until resume does.
func (mg *Manager) Compose(ctx context.Context, in ChainUpIn, from string) error {
	// Refused before the machine is started, so a bad step name does not leave
	// a half-entered machine behind.
	if _, err := mg.stageFor(from); err != nil {
		return err
	}
	mg.stopAfter = ""
	if in.Stage == UpDeploy {
		mg.stopAfter = stepDeploy
	}
	if err := mg.m.Start(ctx, mg.stopped); err != nil {
		return err
	}
	if err := mg.m.Send(ctx, Compose{Request: in, From: from}); err != nil {
		return err
	}
	if mg.failure != nil {
		return mg.failure
	}
	return nil
}

// Tree is the machine's state tree, for a test to compare against the design.
func (mg *Manager) Tree() string { return mg.m.Tree() }

// Steps is one line per stage that finished, in order, for a caller to print.
func (mg *Manager) Steps() []string { return mg.steps }

// note records what a finished stage reported.
func (mg *Manager) note(step, detail string) {
	mg.steps = append(mg.steps, step+": "+detail)
}

// fail records a stage's failure and tells the machine about it.
//
// One place, because every stage says it the same way: wrap with the step's
// name so the message reads as the command a person typed, write it into the
// record so a reader of the workspace finds it, and leave the message that
// moves the machine to failed.
func (mg *Manager) fail(m *statemachine.Machine, step string, cause error) {
	err := fmt.Errorf("chainsetup: chain up: %s: %w", step, cause)
	mg.markFailed(step, err)
	m.SendSelf(stageFailed{Step: step, Err: err})
}

// markFailed writes a failed stage into the record.
//
// Best effort and silent: this runs while a composition is already failing, and
// a second error about the bookkeeping would bury the one the operator came for.
func (mg *Manager) markFailed(step string, cause error) {
	ws, err := Open(mg.ws.Dir(), mg.d.Clock)
	if err != nil {
		return
	}
	ws.MarkStepFailed(step, cause)
	_ = ws.Save()
}

// stageFor is the state that runs the named step, or the first when unnamed.
func (mg *Manager) stageFor(step string) (stage, error) {
	if step == "" {
		return mg.stages[0], nil
	}
	for _, st := range mg.stages {
		if st.step() == step {
			return st, nil
		}
	}
	return nil, fmt.Errorf("chainsetup: no stage is named %q", step)
}

// after is the state to move to once step has finished: the next stage, or
// composed when the request asked to stop here, or ready when there are no more.
func (mg *Manager) after(step string) statemachine.State {
	if step == mg.stopAfter {
		return mg.composed
	}
	for i, st := range mg.stages {
		if st.step() == step {
			if i+1 < len(mg.stages) {
				return mg.stages[i+1]
			}
			return mg.ready
		}
	}
	return mg.ready
}

// recordPath writes where the machine is into the workspace's record.
//
// Through a fresh open, not the workspace this Manager holds. Every step opens
// the workspace again to save what it did, so the copy held across the whole
// run is behind theirs, and saving it would overwrite what they wrote.
//
// A failure here does not fail the composition. The composition is the work and
// this is a note about it — but it is said out loud, because a position nobody
// recorded is exactly what makes a later resume start in the wrong place.
func (mg *Manager) recordPath(s statemachine.State) {
	path := mg.m.Path(s)
	ws, err := Open(mg.ws.Dir(), mg.d.Clock)
	if err == nil {
		ws.SetStatePath(path)
		err = ws.Save()
	}
	if err != nil {
		mg.d.Logf("could not record the composition's position (%s): %v", path, err)
	}
}
