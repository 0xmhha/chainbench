package chainsetup

import (
	"context"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// What a launch is doing, phase by phase.
const (
	nameLaunchingPhase      statemachine.StateName = "LaunchingPhase"
	nameRunningPhaseActions statemachine.StateName = "RunningPhaseActions"
	nameRecordingRun        statemachine.StateName = "RecordingRun"
)

// launching starts the network in the phases the family declares.
//
// A wbft network declares one phase and this is the loop it always was. A
// wemix network starts its producer alone so the etcd cluster can form, and the
// bootstrap runs in the gap before the rest join — launching everything at once
// produced a network that came up and never agreed on anything.
//
// The phases are walked as states rather than as a loop, so the position says
// what the launch was doing and how far it got. A launch that dies in the third
// join has been through LaunchingPhase three times; the record used to hold
// "start" and nothing else.
type launching struct {
	statemachine.Base
	mg *Manager

	bin     string
	phases  []registry.Phase
	at      int
	started int

	phase     *launchingPhase
	actions   *runningPhaseActions
	recording *recordingRun
}

// newLaunching builds the stage and the three things a launch does.
func newLaunching(mg *Manager) *launching {
	s := &launching{mg: mg}
	s.phase = &launchingPhase{parent: s}
	s.actions = &runningPhaseActions{parent: s}
	s.recording = &recordingRun{parent: s}
	return s
}

// Name says what this state is called.
func (launching) Name() statemachine.StateName { return nameLaunching }

// step is which of the composition's steps this state runs.
func (launching) step() string { return stepStart }

// leafStates is what a launch does, in the order the tree shows them.
func (s *launching) leafStates() []statemachine.State {
	return []statemachine.State{s.phase, s.actions, s.recording}
}

// Enter makes the checks a launch makes before it starts anything.
func (s *launching) Enter(ctx context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	s.at, s.started = 0, 0
	type plan struct {
		bin    string
		phases []registry.Phase
	}
	p, err := InWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (plan, error) {
		bin, phases, perr := ws.LaunchPlan(ctx, s.mg.request.Binary)
		return plan{bin, phases}, perr
	})
	if err != nil {
		s.mg.fail(m, stepStart, err)
		return nil
	}
	s.bin, s.phases = p.bin, p.phases
	m.SendSelf(launchPlanned{Phases: len(p.phases)})
	return nil
}

// Process walks the phases: launch one, run its actions if it has any, then the
// next — and when there are none left, the wrap-up.
func (s *launching) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	switch e := msg.(type) {
	case launchPlanned:
		m.TransitionTo(s.next(m))
		return true, nil
	case phaseLaunched:
		s.started += e.Started
		if len(s.phases[s.at].Actions) > 0 {
			m.TransitionTo(s.actions)
			return true, nil
		}
		s.at++
		m.TransitionTo(s.next(m))
		return true, nil
	case phaseActionsDone:
		s.at++
		m.TransitionTo(s.next(m))
		return true, nil
	}
	return false, nil
}

// next is the phase state again, or the wrap-up when the phases are done.
//
// Going back to the same state is a move, not a no-op: leaving it and entering
// it is what starts the next phase. That is why a transition to the current
// state exits and enters rather than being skipped.
func (s *launching) next(*statemachine.Machine) statemachine.State {
	if s.at < len(s.phases) {
		return s.phase
	}
	return s.recording
}

// launchingPhase starts one phase's nodes.
type launchingPhase struct {
	statemachine.Base
	parent *launching
}

// Name says what this state is called.
func (launchingPhase) Name() statemachine.StateName { return nameLaunchingPhase }

// Enter starts this phase and says how many nodes went up.
func (l *launchingPhase) Enter(ctx context.Context, m *statemachine.Machine) error {
	s := l.parent
	s.mg.recordPath(l)
	phase := s.phases[s.at]
	started, err := InWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (int, error) {
		return ws.StartPhase(ctx, s.bin, phase)
	})
	if err != nil {
		s.mg.fail(m, stepStart, err)
		return nil
	}
	m.SendSelf(phaseLaunched{Started: started})
	return nil
}

// runningPhaseActions runs what a phase declares after its nodes are up.
type runningPhaseActions struct {
	statemachine.Base
	parent *launching
}

// Name says what this state is called.
func (runningPhaseActions) Name() statemachine.StateName { return nameRunningPhaseActions }

// Enter runs this phase's actions.
func (l *runningPhaseActions) Enter(ctx context.Context, m *statemachine.Machine) error {
	s := l.parent
	s.mg.recordPath(l)
	phase := s.phases[s.at]
	_, err := InWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (struct{}, error) {
		return struct{}{}, ws.RunPhaseActions(ctx, s.bin, phase)
	})
	if err != nil {
		s.mg.fail(m, stepStart, err)
		return nil
	}
	m.SendSelf(phaseActionsDone{})
	return nil
}

// recordingRun is what is left after the last phase: the binary this network
// runs, the step marked done, and the run written down.
//
// A state of its own because it is work, and work happens on the way into a
// state. It also makes one thing visible that used to be a clause appended to a
// line of output: a network that came up and whose run record could not be
// written is a position, not a sentence.
type recordingRun struct {
	statemachine.Base
	parent *launching
}

// Name says what this state is called.
func (recordingRun) Name() statemachine.StateName { return nameRecordingRun }

// Enter records the run and reports the stage finished.
func (l *recordingRun) Enter(ctx context.Context, m *statemachine.Machine) error {
	s := l.parent
	s.mg.recordPath(l)
	detail, err := InWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (string, error) {
		return ws.FinishLaunch(ctx, s.bin, s.started)
	})
	if err != nil {
		s.mg.fail(m, stepStart, err)
		return nil
	}
	m.SendSelf(nodesLaunched{Detail: detail})
	return nil
}
