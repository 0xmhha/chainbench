package chainsetup

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/hardfork"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/preflight"
	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

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
	nameChain                     statemachine.StateName = "CHAIN"
	nameChainIdle                 statemachine.StateName = "CHAIN_IDLE"
	nameChainBuildUp              statemachine.StateName = "CHAIN_BUILD_UP"
	nameChainOpenWorkspace        statemachine.StateName = "CHAIN_OPEN_WORKSPACE"
	nameChainBuildNodeTable       statemachine.StateName = "CHAIN_BUILD_NODE_TABLE"
	nameChainEnsureKeys           statemachine.StateName = "CHAIN_ENSURE_KEYS"
	nameChainBuildGenesis         statemachine.StateName = "CHAIN_BUILD_GENESIS"
	nameChainBuildNodeConfig      statemachine.StateName = "CHAIN_BUILD_NODE_CONFIG"
	nameChainBuildNodeCommand     statemachine.StateName = "CHAIN_BUILD_NODE_COMMAND"
	nameChainDeployNodes          statemachine.StateName = "CHAIN_DEPLOY_NODES"
	nameChainInitNodes            statemachine.StateName = "CHAIN_INIT_NODES"
	nameChainLaunchNodes          statemachine.StateName = "CHAIN_LAUNCH_NODES"
	nameChainBuildUpStoppedAtStep statemachine.StateName = "CHAIN_BUILD_UP_STOPPED_AT_STEP"
	nameChainReady                statemachine.StateName = "CHAIN_READY"
	nameChainFailed               statemachine.StateName = "CHAIN_FAILED"
	nameChainOp                   statemachine.StateName = "CHAIN_OP"
	nameChainStopped              statemachine.StateName = "CHAIN_STOPPED"
	nameChainRemoved              statemachine.StateName = "CHAIN_REMOVED"
)

// stageOrder pairs each composition step with the state that runs it.
//
// The steps are UpStepNames and manager_test.go holds them to it, so the order
// a record is read in and the order the machine walks cannot drift apart.
var stageOrder = []struct {
	step string
	name statemachine.StateName
}{
	{stepNew, nameChainOpenWorkspace},
	{stepPlace, nameChainBuildNodeTable},
	{stepKeys, nameChainEnsureKeys},
	{stepGenesis, nameChainBuildGenesis},
	{stepConfig, nameChainBuildNodeConfig},
	{stepBuild, nameChainBuildNodeCommand},
	{stepDeploy, nameChainDeployNodes},
	{stepInit, nameChainInitNodes},
	{stepStart, nameChainLaunchNodes},
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
	d  Deps
	ws *Workspace
	m  *statemachine.Machine

	// request is what this run was asked to compose. A stage reads the parts it
	// needs from it; it is stored once, when the request arrives.
	request ChainUpIn
	// stopAfter is the last step this run performs, or "" to run them all.
	stopAfter string
	// reuse is what was running before this composition started, for the
	// reconciliation to compare against. Its presence is what makes a run a
	// reuse-if-matching one.
	reuse   ReuseSnapshot
	reusing bool
	// only is the one step a run was asked for, and "" for a whole composition.
	// A step asked for by name stops when it is done rather than walking on.
	only string
	// steps is one "step: detail" line per stage that finished, in order. It is
	// what a caller prints, and it is kept even when the run then failed,
	// because how far it got is the first thing a reader wants.
	steps []string
	// failure is why the composition stopped, written just before the move to
	// failed so that state can report it once.
	failure error
	// compare asks the target what it holds against the request. It is a field
	// so a test can hand the comparison a verdict without a live network.
	compare func(context.Context, Deps, ChainUpIn) preflight.Decision

	stopped     *stoppedState
	comparing   *comparing
	composing   *composingState
	reconciling *reconciling
	stages      []stage
	composed    *composedState
	ready       *readyState
	op          *opState
	ops         []operation
	opStopped   *opResult
	opRemoved   *opResult
	// operating is the operation this run was asked for, which is what says
	// where the machine rests once it is done.
	operating operation

	stopping     *stopping
	restarting   *restarting
	swapping     *swapping
	hardforking  *hardforking
	crossingFork *crossingFork
	removing     *removing
	failed       *failedState
}

// NewManager builds a composition machine over an open, held workspace.
//
// The Add calls below are the tree. They are written as the tree so the shape
// can be read rather than reconstructed, which is what the reference does with
// its own addState block.
func NewManager(d Deps, ws *Workspace) *Manager {
	mg := &Manager{d: d, ws: ws, m: statemachine.New("composition", nil), compare: compareWorkspace}

	root := &compositionState{mg: mg}
	mg.stopped = &stoppedState{mg: mg}
	mg.reconciling = &reconciling{mg: mg}
	mg.comparing = newComparing(mg)
	mg.stopping = &stopping{mg: mg}
	mg.restarting = &restarting{mg: mg}
	mg.swapping = &swapping{mg: mg}
	mg.hardforking = &hardforking{mg: mg}
	mg.crossingFork = newCrossingFork(mg)
	mg.removing = &removing{mg: mg}
	mg.ops = []operation{
		mg.stopping, mg.restarting, mg.swapping,
		mg.hardforking, mg.crossingFork, mg.removing,
	}
	mg.composing = &composingState{mg: mg}
	mg.composed = &composedState{mg: mg}
	mg.ready = &readyState{mg: mg}
	mg.op = &opState{mg: mg}
	mg.opStopped = &opResult{mg: mg, name: nameChainStopped}
	mg.opRemoved = &opResult{mg: mg, name: nameChainRemoved}
	mg.failed = &failedState{mg: mg}
	// One state per stage, in the order a composition runs them.
	mg.stages = []stage{
		&openingWorkspace{mg: mg},
		&buildingNodeTable{mg: mg},
		newEnsuringKeys(mg),
		newBuildingGenesis(mg),
		&buildingNodeConfig{mg: mg},
		&buildingNodeCommand{mg: mg},
		newDeployingInputs(mg),
		&initializingDatadirs{mg: mg},
		newLaunching(mg),
	}

	// Every state says what it handles and sends, and the machine holds it to
	// that (design-v3 state-machine-06 §5).
	mg.m.RequireContracts()
	mg.m.Add(root, nil)
	mg.m.Add(mg.stopped, root)
	mg.m.Add(mg.comparing, root)
	for _, leaf := range mg.comparing.leafStates() {
		mg.m.Add(leaf, mg.comparing)
	}
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
		// The comparison stands where it happens: after the keys, before the
		// genesis writes anything.
		if st.step() == stepKeys {
			mg.m.Add(mg.reconciling, mg.composing)
		}
	}
	mg.m.Add(mg.composed, root)
	mg.m.Add(mg.ready, root)
	// Operations are not children of ready: stopping a network does not leave
	// it ready, and a parent is what a state is read as being part of.
	mg.m.Add(mg.op, root)
	for _, op := range mg.ops {
		mg.m.Add(op, mg.op)
		// An operation that goes somewhere on the way says so, and its moments
		// go in right under it — the same shape a stage with more than one way
		// of working has.
		if branch, ok := op.(interface {
			leafStates() []statemachine.State
		}); ok {
			for _, leaf := range branch.leafStates() {
				mg.m.Add(leaf, op)
			}
		}
	}
	mg.m.Add(mg.opStopped, root)
	mg.m.Add(mg.opRemoved, root)
	mg.m.Add(mg.failed, root)

	// A message no state handles ends the composition the way a failed stage
	// does, so the caller reads one reason from one place.
	mg.m.OnUnhandled(func(msg statemachine.Message, at statemachine.StateName) statemachine.Message {
		return stageFailed{Step: "machine", Err: fmt.Errorf("chainsetup: %w: %s in %s",
			statemachine.ErrUnhandled, WhatName(msg.What()), at)}
	})
	return mg
}

// ends is where an entry point may leave the machine. Returning from anywhere
// else would report a stalled walk as a success.
func (mg *Manager) ends(ctx context.Context, states ...statemachine.State) error {
	if mg.failure != nil {
		return mg.failure
	}
	return mg.m.RequireAt(states...)
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
	return mg.ends(ctx, mg.ready, mg.composed, mg.failed)
}

// Step runs one composition step by name, on a workspace that has got that far.
//
// It is what a standalone `chain <step>` command does. The machine is born at
// rest every time, because one command is one process, so where a composition
// had got to is read from its record rather than remembered.
func (mg *Manager) Step(ctx context.Context, step string, in ChainUpIn) (string, error) {
	if _, err := mg.stageFor(step); err != nil {
		return "", err
	}
	mg.only = step
	if err := mg.m.Start(ctx, mg.stopped); err != nil {
		return "", err
	}
	if err := mg.m.Send(ctx, RunStep{Name: step, Request: in}); err != nil {
		return "", err
	}
	if err := mg.ends(ctx, mg.composed, mg.failed); err != nil {
		return "", err
	}
	if len(mg.steps) == 0 {
		return "", fmt.Errorf("chainsetup: %s reported nothing", step)
	}
	_, detail, _ := strings.Cut(mg.steps[len(mg.steps)-1], ": ")
	return detail, nil
}

// ResumeStep is the step a composition should be resumed at.
//
// The recorded position is the better answer where it is decisive: a stage
// writes its own path on the way in, so a run that died mid-stage names the
// stage that did not finish, and re-entering it does that stage over. Working
// the same thing out by walking UpStepNames for the first rung not done is a
// second account of the position kept somewhere else, and the two have
// disagreed.
//
// Two positions are deliberate stops rather than places a run is in. Ready is
// finished. Composed is a run that stopped where it was told — after a
// --stage=deploy, or after one step run by hand — and what should follow is the
// step map's to answer, because the request rather than the position decides
// whether there is more to do.
func (w *Workspace) ResumeStep() string {
	switch w.state.StatePath {
	case string(nameChain) + "/" + string(nameChainReady),
		string(nameChain) + "/" + string(nameChainStopped),
		string(nameChain) + "/" + string(nameChainRemoved):
		return ""
	case "", string(nameChain) + "/" + string(nameChainIdle),
		string(nameChain) + "/" + string(nameChainBuildUpStoppedAtStep):
		return w.FirstUndone()
	}
	for _, part := range strings.Split(w.state.StatePath, "/") {
		for _, st := range stageOrder {
			if part == string(st.name) {
				return st.step
			}
		}
	}
	return w.FirstUndone()
}

// stepIsDue reports whether every step this one needs has already run.
//
// It reads the recorded steps rather than the state fields they leave behind. A
// field can be non-empty because something else filled it, and a step that
// half-ran leaves exactly that: state that looks composed and was not.
//
// The same table the step bodies read, not a second copy. While paths that do
// not run on this machine still exist, the check lives in two places, and two
// places reading one table cannot disagree about the answer.
func (mg *Manager) stepIsDue(step string) error {
	ws, err := Open(mg.ws.Dir(), mg.d.Clock)
	if err != nil {
		return err
	}
	done := ws.State().Steps
	for _, need := range composeNeeds[step] {
		if _, ok := done[need]; !ok {
			return fmt.Errorf("chainsetup: %s: %s has not run — run `chain %s` first", step, need, need)
		}
	}
	return nil
}

// ReuseFrom tells this composition to hold what is already running against what
// it would build, before it builds anything.
//
// The snapshot is taken by the caller, before the machine starts, because the
// compose steps reset the node table: by the time a state could take it, what
// was on the target is already gone.
func (mg *Manager) ReuseFrom(snap ReuseSnapshot) {
	mg.reuse, mg.reusing = snap, true
}

// ComposeComparing composes the network the request declares, reusing what is
// on the target when the comparison says it can.
//
// It stops at the hand-over rather than at Ready: whether the network is
// producing is the readiness gate's question, and the gate belongs to whoever
// owns the monitor.
func (mg *Manager) ComposeComparing(ctx context.Context, in ChainUpIn) error {
	mg.request = in
	mg.stopAfter = ""
	if in.Stage == UpDeploy {
		mg.stopAfter = stepDeploy
	}
	if err := mg.m.Start(ctx, mg.stopped); err != nil {
		return err
	}
	if err := mg.m.Send(ctx, ComposeComparing{Request: in}); err != nil {
		return err
	}
	return mg.ends(ctx, mg.ready, mg.composed, mg.failed)
}

// Operate runs one operation on a composed network.
//
// The arguments are already on the state that will run it; this says which one,
// and the machine says whether a network in this condition may be asked.
func (mg *Manager) Operate(ctx context.Context, op operation) (string, error) {
	mg.operating = op
	if err := mg.m.Start(ctx, mg.stopped); err != nil {
		return "", err
	}
	if err := mg.m.Send(ctx, Operate{Name: op.Name()}); err != nil {
		return "", err
	}
	if err := mg.ends(ctx, mg.ready, mg.opStopped, mg.opRemoved, mg.failed); err != nil {
		return "", err
	}
	if len(mg.steps) == 0 {
		return "", nil
	}
	_, detail, _ := strings.Cut(mg.steps[len(mg.steps)-1], ": ")
	return detail, nil
}

// Stop takes every running node down.
func (mg *Manager) Stop(ctx context.Context) (string, error) {
	return mg.Operate(ctx, mg.stopping)
}

// Remove takes the composed network's data away.
func (mg *Manager) Remove(ctx context.Context) (string, error) {
	return mg.Operate(ctx, mg.removing)
}

// RestartNode bounces one node.
func (mg *Manager) RestartNode(ctx context.Context, index int) (string, error) {
	mg.restarting.index = index
	return mg.Operate(ctx, mg.restarting)
}

// Swap replaces one node's binary.
func (mg *Manager) Swap(ctx context.Context, opts SwapNodeOpts) (string, error) {
	mg.swapping.opts = opts
	return mg.Operate(ctx, mg.swapping)
}

// Hardfork swaps the network's binary at a fork block, keeping node data.
func (mg *Manager) Hardfork(ctx context.Context, plan hardfork.SwapPlan, binary string) (node.NodeSet, error) {
	mg.hardforking.plan, mg.hardforking.binary = plan, binary
	if _, err := mg.Operate(ctx, mg.hardforking); err != nil {
		return node.NodeSet{}, err
	}
	return mg.hardforking.nodes, nil
}

// CrossFork waits for the network to cross the fork it is planned for.
func (mg *Manager) CrossFork(ctx context.Context, opts CrossForkOpts) (StepOut, error) {
	mg.crossingFork.opts = opts
	_, err := mg.Operate(ctx, mg.crossingFork)
	return mg.crossingFork.out, err
}

// operationNamed is the state that runs the named operation.
func (mg *Manager) operationNamed(name statemachine.StateName) (operation, error) {
	for _, op := range mg.ops {
		if op.Name() == name {
			return op, nil
		}
	}
	return nil, fmt.Errorf("chainsetup: no operation is named %q", name)
}

// mayOperate reports whether a composed network in this condition may be asked
// to do this.
//
// It asks the same verbNeeds table the bodies ask, not a second copy. What an
// operation needs — a node table that exists, a node that is stopped, an argv
// that was recorded — is a fact about the network rather than a position in a
// tree, so the position cannot answer it and does not pretend to.
func (mg *Manager) mayOperate(verb string) error {
	ws, err := Open(mg.ws.Dir(), mg.d.Clock)
	if err != nil {
		return err
	}
	ws.SetEnv(mg.d.Env)
	ws.SetDriver(mg.d.Driver)
	return ws.allow(verb)
}

// operated runs an operation's body and reports what it did.
//
// One place, because the six do the same three things around their own call:
// record where the machine is, run it in the workspace, and say what happened.
func (mg *Manager) operated(m *statemachine.Machine, op operation, fn func(*Workspace) (string, error)) {
	mg.recordPath(op)
	detail, err := InWorkspace(mg.d, mg.ws.Dir(), fn)
	if err != nil {
		m.SendSelf(stageFailed{Step: lower(op.verb()), Err: err})
		return
	}
	mg.note(lower(op.verb()), detail)
	m.SendSelf(operationDone{})
}

// Decision is what the comparison decided, as a report prints it.
func (mg *Manager) Decision() string {
	var none preflight.Verdict
	if mg.comparing.decision.Verdict == none {
		return ""
	}
	return mg.comparing.decision.String()
}

// At is where the machine ended, as a path, for a caller that reports it.
func (mg *Manager) At() string { return mg.m.Path(mg.m.Current()) }

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

// failLaunch fails the start stage after stopping the nodes this launch
// started — every node with a pid now that was not in wasRunning.
//
// A composition that failed is not a network anyone will use: the caller gets
// an error and no way to stop what came up, so the nodes that did come up are
// taken down here, where it is still known which ones they are. When the
// rollback itself fails, both are said — the launch's error first, because it
// is the one the operator came for.
func (mg *Manager) failLaunch(ctx context.Context, m *statemachine.Machine, wasRunning map[int]bool, cause error) {
	detail, rerr := InWorkspace(mg.d, mg.ws.Dir(), func(ws *Workspace) (string, error) {
		return ws.RollBackLaunch(ctx, wasRunning)
	})
	if rerr != nil {
		cause = fmt.Errorf("%w (and %v)", cause, rerr)
	} else {
		mg.note("rollback", detail)
	}
	mg.fail(m, stepStart, cause)
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
	// The comparison stands between the keys and the genesis: the last moment
	// what is on the target is still there to compare.
	if step == stepKeys && mg.reusing {
		return mg.reconciling
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
