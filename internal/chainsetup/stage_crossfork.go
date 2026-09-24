package chainsetup

import (
	"context"

	"github.com/0xmhha/chainbench/internal/core/statemachine"
)

// The three moments crossing a fork goes through.
const (
	nameChainOpCrossForkAwaitBoundary statemachine.StateName = "CHAIN_OP_CROSS_FORK_AWAIT_BOUNDARY"
	nameChainOpCrossForkHandOver      statemachine.StateName = "CHAIN_OP_CROSS_FORK_HAND_OVER"
	nameChainOpCrossForkConfirm       statemachine.StateName = "CHAIN_OP_CROSS_FORK_CONFIRM"
)

// crossingFork waits for the network to cross the fork it is planned for.
//
// It is the only operational verb that waits on the chain rather than on a
// process, and the only one that goes anywhere on the way: the chain reaches
// the block before the fork, the successors take production over there, and the
// chain is held to having crossed on the build that replaces the old one. Those
// three are what somebody looking at a stopped crossing needs to tell apart. A
// head that never arrived is a network to look at; a hand-over that failed is a
// launch to look at.
//
// They used to be a list the verb appended to as it went and returned beside
// its result. The list said what a position says, kept by hand, and it lied
// about the one case it could not walk: a network already across was reported
// as having passed all three moments, which no run of it ever did. Now that
// network goes straight to Crossed, and the path is the answer.
type crossingFork struct {
	statemachine.Base
	mg   *Manager
	opts CrossForkOpts

	// head is the block the network stood at when the hand-over began, and
	// already says it was across before anybody asked. Both are read by the
	// last moment, which writes the sentence the step reports.
	head    int64
	already bool

	// out is what the crossing reported, for the caller that asked for it.
	out StepOut

	beforeFork  *beforeFork
	handingOver *handingOver
	crossed     *crossedFork
}

// newCrossingFork builds the operation and the three moments it goes through.
func newCrossingFork(mg *Manager) *crossingFork {
	s := &crossingFork{mg: mg}
	s.beforeFork = &beforeFork{parent: s}
	s.handingOver = &handingOver{parent: s}
	s.crossed = &crossedFork{parent: s}
	return s
}

// Name says what this state is called.
func (crossingFork) Name() statemachine.StateName { return nameChainOpCrossFork }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (crossingFork) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: []statemachine.What{eventForkStandingRead, eventForkBoundaryReached, eventProductionHandedOver}, Emits: []statemachine.What{eventForkStandingRead, eventStageFailed}}
}

// verb is the name verbNeeds declares this operation's conditions under. It is
// the first thing the crossing does, and what it may do is what that asks.
func (crossingFork) verb() string { return "ForkStanding" }

// leafStates is the moments this crossing goes through, in the order the tree
// shows them.
func (s *crossingFork) leafStates() []statemachine.State {
	return []statemachine.State{s.beforeFork, s.handingOver, s.crossed}
}

// Enter reads whether there is a crossing left to do.
func (s *crossingFork) Enter(_ context.Context, m *statemachine.Machine) error {
	s.mg.recordPath(s)
	s.head, s.already, s.out = 0, false, StepOut{}
	standing, err := InWorkspace(s.mg.d, s.mg.ws.Dir(), func(ws *Workspace) (ForkStanding, error) {
		return ws.ForkStanding()
	})
	if err != nil {
		s.mg.fail(m, crossForkStep, err)
		return nil
	}
	// Written out rather than converted from ForkStanding: the two match by
	// accident, and a conversion would tie this message's shape to a type it
	// only happens to agree with today.
	m.SendSelf(forkStandingRead{Crossed: standing.Crossed}) //nolint:staticcheck // see above
	return nil
}

// Process walks the three moments.
//
// The moves are here rather than in each moment because what follows one is
// this operation's shape, not the moment's: the same state ends a crossing that
// was walked and one that had already happened.
func (s *crossingFork) Process(_ context.Context, m *statemachine.Machine, msg statemachine.Message) (bool, error) {
	next, ok := s.nextMoment(msg)
	if !ok {
		return false, nil
	}
	m.TransitionTo(next)
	return true, nil
}

// nextMoment is the moment that follows this message, and whether this crossing
// knows the message at all.
//
// It is apart from Process so the shape can be asked without a network to walk
// it on: which moment follows which is a decision, and a decision that can only
// be checked by crossing a real fork is one nobody checks.
func (s *crossingFork) nextMoment(msg statemachine.Message) (statemachine.State, bool) {
	switch e := msg.(type) {
	case forkStandingRead:
		s.already = e.Crossed
		if e.Crossed {
			return s.crossed, true
		}
		return s.beforeFork, true
	case forkBoundaryReached:
		s.head = e.Head
		return s.handingOver, true
	case productionHandedOver:
		return s.crossed, true
	}
	return nil, false
}

// beforeFork is the chain standing at the block the hand-over begins from.
type beforeFork struct {
	statemachine.Base
	parent *crossingFork
}

// Name says what this state is called.
func (beforeFork) Name() statemachine.StateName { return nameChainOpCrossForkAwaitBoundary }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (beforeFork) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventForkBoundaryReached, eventStageFailed}}
}

// Enter brings the network to the point where it can be moved.
func (s *beforeFork) Enter(ctx context.Context, m *statemachine.Machine) error {
	mg := s.parent.mg
	mg.recordPath(s)
	head, err := InWorkspace(mg.d, mg.ws.Dir(), func(ws *Workspace) (int64, error) {
		return ws.ForkMoment(ctx, s.parent.opts)
	})
	if err != nil {
		mg.fail(m, crossForkStep, err)
		return nil
	}
	m.SendSelf(forkBoundaryReached{Head: head})
	return nil
}

// handingOver is the pre-fork nodes down and the build that seals after the
// fork coming up in their place.
type handingOver struct {
	statemachine.Base
	parent *crossingFork
}

// Name says what this state is called.
func (handingOver) Name() statemachine.StateName { return nameChainOpCrossForkHandOver }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (handingOver) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventProductionHandedOver, eventStageFailed}}
}

// Enter hands production over.
func (s *handingOver) Enter(ctx context.Context, m *statemachine.Machine) error {
	mg := s.parent.mg
	mg.recordPath(s)
	_, err := InWorkspace(mg.d, mg.ws.Dir(), func(ws *Workspace) (struct{}, error) {
		return struct{}{}, ws.HandOverFork(ctx)
	})
	if err != nil {
		mg.fail(m, crossForkStep, err)
		return nil
	}
	m.SendSelf(productionHandedOver{})
	return nil
}

// crossedFork is the fork behind the network.
//
// Two ways in, and they reach one state because they are one fact: the
// successors produce and the chain is past the fork. What differs is only what
// is left to check, which is nothing for a network that was across before
// anybody asked.
type crossedFork struct {
	statemachine.Base
	parent *crossingFork
}

// Name says what this state is called.
func (crossedFork) Name() statemachine.StateName { return nameChainOpCrossForkConfirm }

// Contract is what this state handles and sends (design-v3 state-machine-06 §5).
func (crossedFork) Contract() statemachine.Contract {
	return statemachine.Contract{Accepts: nil, Emits: []statemachine.What{eventOperationDone, eventStageFailed}}
}

// Enter confirms the crossing, records the step and ends the operation.
func (s *crossedFork) Enter(ctx context.Context, m *statemachine.Machine) error {
	mg := s.parent.mg
	mg.recordPath(s)
	detail, err := InWorkspace(mg.d, mg.ws.Dir(), func(ws *Workspace) (string, error) {
		if s.parent.already {
			return ws.ReportAlreadyCrossed()
		}
		return ws.ConfirmCrossing(ctx, s.parent.head)
	})
	if err != nil {
		mg.fail(m, crossForkStep, err)
		return nil
	}
	s.parent.out = StepOut{Detail: detail}
	mg.note(crossForkStep, detail)
	m.SendSelf(operationDone{})
	return nil
}
