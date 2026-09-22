package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// StepRunner runs one composition step by name.
//
// The bodies live in the verb layer, which imports this package, so they are
// handed in rather than reached for. It is the same seam the composition
// already had: the runner closure the old machine's handlers were built from.
type StepRunner func(ctx context.Context, step string) error

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
	{"new", nameOpeningWorkspace},
	{"place", nameBuildingNodeTable},
	{"keys", nameEnsuringKeys},
	{"genesis", nameBuildingGenesis},
	{"config", nameBuildingNodeConfig},
	{"build", nameBuildingNodeCommand},
	{"deploy", nameDeployingInputs},
	{"init", nameInitializingDatadirs},
	{"start", nameLaunching},
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

	// stopAfter is the last step this run performs, or "" to run them all.
	stopAfter string
	// failure is why the composition stopped, written just before the move to
	// failed so that state can report it once.
	failure error

	stopped   *stoppedState
	composing *composingState
	leaves    []*legacyStage
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
	for _, s := range stageOrder {
		mg.leaves = append(mg.leaves, &legacyStage{mg: mg, step: s.step, name: s.name})
	}

	mg.m.Add(root, nil)
	mg.m.Add(mg.stopped, root)
	mg.m.Add(mg.composing, root)
	for _, leaf := range mg.leaves {
		mg.m.Add(leaf, mg.composing)
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
func (mg *Manager) Compose(ctx context.Context, in NetUpIn, from string) error {
	// Refused before the machine is started, so a bad step name does not leave
	// a half-entered machine behind.
	if _, err := mg.leafFor(from); err != nil {
		return err
	}
	mg.stopAfter = ""
	if in.Stage == UpDeploy {
		mg.stopAfter = "deploy"
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

// leafFor is the state that runs the named step, or the first when unnamed.
func (mg *Manager) leafFor(step string) (*legacyStage, error) {
	if step == "" {
		return mg.leaves[0], nil
	}
	for _, leaf := range mg.leaves {
		if leaf.step == step {
			return leaf, nil
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
	for i, leaf := range mg.leaves {
		if leaf.step == step {
			if i+1 < len(mg.leaves) {
				return mg.leaves[i+1]
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
