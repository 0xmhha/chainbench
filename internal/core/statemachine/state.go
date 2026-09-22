package statemachine

import "context"

// StateName is what a state is called, and how the machine tells states apart.
//
// It must be unique within one machine: [Machine.Add] refuses a repeat, because
// two states answering to one name make the log and the recorded path ambiguous
// about which one a run was in.
type StateName string

// State is one state of a machine.
//
// The three methods divide the work. Enter starts this state's own work and
// says nothing about what comes next. Process reads a message and, if this
// state is the one that should decide, names the next state. Exit gives back
// what Enter took. A state that implements only some of them embeds [Base].
type State interface {
	// Name is what this state is called. It must be stable: the machine keys
	// its tree on it and a run's record keeps it.
	Name() StateName

	// Enter starts this state's work. It does not return the result — the
	// result comes back as a message, either one this state leaves itself with
	// [Machine.SendSelf] or one that arrives later. Calling
	// [Machine.TransitionTo] here is refused.
	Enter(ctx context.Context, m *Machine) error

	// Exit undoes what Enter armed: a timer cancelled, a file closed, a record
	// saved. Calling [Machine.TransitionTo] here is refused.
	Exit(ctx context.Context, m *Machine) error

	// Process handles a message. It reports whether this state handled it; an
	// unhandled message is offered to the parent, and so on up. A state that
	// handles a message may name the next state with [Machine.TransitionTo],
	// once.
	Process(ctx context.Context, m *Machine, msg Message) (handled bool, err error)
}

// Base is the empty implementation, to embed in a state that does not need all
// three methods.
//
// Name is deliberately not among them. A state with no name would still
// compile and would then be indistinguishable in the tree, the log and the
// record, so every state has to say what it is called.
type Base struct{}

// Enter does nothing.
func (Base) Enter(context.Context, *Machine) error { return nil }

// Exit does nothing.
func (Base) Exit(context.Context, *Machine) error { return nil }

// Process handles nothing, so every message is offered to the parent.
func (Base) Process(context.Context, *Machine, Message) (bool, error) { return false, nil }
