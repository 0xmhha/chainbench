package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// The composition's states.
//
// Every one of them is small on purpose: a state says what it does on the way
// in, and what one message means once it is there. Everything else is the
// stage's work, which lives where it always did.

// stage is a state that runs one of the composition's steps.
//
// The Manager keeps them in order and asks each which step it is, so the walk
// and the record's step names stay one list while the states behind them are
// replaced one at a time.
type stage interface {
	statemachine.State
	step() string
}

// compositionState is the root. It handles nothing, so a message no state below
// wants is recorded as unhandled rather than mistaken for something.
type compositionState struct{ statemachine.Base }

// Name says what this state is called.
func (compositionState) Name() statemachine.StateName { return nameComposition }

// stoppedState is where a machine starts and where a cleared failure returns to.
type stoppedState struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (stoppedState) Name() statemachine.StateName { return nameStopped }

// Process takes the request and goes to the stage the run begins at.
//
// Choosing the first stage is this state's job rather than the caller's: the
// request is what says where a composition starts, and reading it here is what
// lets the caller send one message instead of computing a starting point.
func (s *stoppedState) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	c, ok := msg.(Compose)
	if !ok {
		return false, nil
	}
	first, err := s.mg.stageFor(c.From)
	if err != nil {
		return true, err
	}
	s.mg.request = c.Request
	m.TransitionTo(first)
	return true, nil
}

// composingState is the stages' parent. It owns what a finished stage means and
// what a failed one means, so no stage has to know what comes after it.
type composingState struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (composingState) Name() statemachine.StateName { return nameComposing }

// Process routes a stage's report.
func (s *composingState) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	switch e := msg.(type) {
	case stageReport:
		step, detail := e.stage()
		s.mg.note(step, detail)
		m.TransitionTo(s.mg.after(step))
		return true, nil
	case stageFailed:
		// The reason is written before the move, so the failed state reads a
		// fact rather than being handed one. Transitions carry no values.
		s.mg.failure = e.Err
		m.TransitionTo(s.mg.failed)
		return true, nil
	}
	return false, nil
}

// composedState is a composition that stopped where it was asked to.
type composedState struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (composedState) Name() statemachine.StateName { return nameComposed }

// Enter records that the composition stopped here.
func (s *composedState) Enter(context.Context, *statemachine.Machine) error {
	s.mg.recordPath(s)
	return nil
}

// readyState is a composition that ran every stage it had.
type readyState struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (readyState) Name() statemachine.StateName { return nameReady }

// Enter records that the composition finished.
func (s *readyState) Enter(context.Context, *statemachine.Machine) error {
	s.mg.recordPath(s)
	return nil
}

// failedState is a composition that stopped because a stage could not finish.
//
// It holds the reason rather than one state per kind of failure. Which failures
// deserve a state of their own is a question about what has to be undone after
// each, and that is answered by running the thing, not by guessing before.
type failedState struct {
	statemachine.Base
	mg *Manager
}

// Name says what this state is called.
func (failedState) Name() statemachine.StateName { return nameFailed }

// Enter records the position, which is the stage that failed.
func (s *failedState) Enter(context.Context, *statemachine.Machine) error {
	s.mg.recordPath(s)
	return nil
}

// Process lets a caller leave the failure, having read it.
//
// Nothing else is accepted, so a command that runs into a failed composition is
// refused by name instead of quietly composing on top of one.
func (s *failedState) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	switch msg.(type) {
	case ClearError:
		s.mg.failure = nil
		m.TransitionTo(s.mg.stopped)
		return true, nil
	}
	return true, fmt.Errorf("chainsetup: the composition failed and has not been cleared: %w", s.mg.failure)
}
