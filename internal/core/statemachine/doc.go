// Package statemachine is a hierarchical state machine that messages drive.
//
// The mechanism is one sentence. Entering a state starts that state's work;
// the work's result comes back as a message; [State.Process] turns that message
// into a move; and the next piece of work is the next state's [State.Enter].
// [State.Exit] takes back what Enter armed. States chain that way, and every
// link in the chain is one message.
//
// What this is NOT: a table of permitted moves. Nothing here holds a list of
// which state may follow which. A state names its own successor by calling
// [Machine.TransitionTo], so the move and the reason for it sit in the same
// place, and a reader looking for "what happens after the genesis is built"
// finds it in the state that built it rather than in a table somewhere else.
//
// # The synchronous model
//
// There is no Looper and no thread. [Machine.Send] is one complete unit: it
// dispatches the message from the current state upward through its parents,
// performs at most one transition, then drains the self queue and the inbox
// until both are empty. When Send returns, the machine is at rest. Re-entering
// Send is refused, because a message half-processed inside another message's
// processing is the thing the queues exist to prevent.
//
// Delayed messages and deferMessage are deliberately absent. Composition is a
// sequence of synchronous calls with one caller, so a state's Enter learns its
// result on the spot and no second message arrives while the first is being
// handled. The day an outside observer's event has to survive a stage it
// arrived during, deferMessage is what to add.
//
// # Why a state cannot move during Enter or Exit
//
// A transition that started another transition halfway through would leave the
// active set describing neither the state it came from nor the one it is going
// to. So [Machine.TransitionTo] is accepted only inside Process, and only once
// per message. A state that wants to move as a consequence of entering leaves
// itself a message with [Machine.SendSelf]: the transition finishes, the new
// state becomes current, and the message is then handled there.
//
// # Up is a queue, down is a call
//
// A child machine reports to its controller with [Machine.Post], which only
// enqueues. A controller drives a child with [Machine.Send], a nested call.
// Two synchronous machines calling Send into each other would be re-entrant,
// and this asymmetry is what keeps them from being.
//
// # Identity
//
// A state is identified by its [StateName], which must be unique within one
// machine. Nothing here compares State values, so a state is free to hold
// fields that Go cannot compare — a func, a slice, a map — and domain code that
// does want to compare states holds them as pointers.
//
// # Errors
//
// A returned error means the machine cannot go on: a state was misused, or
// something outside any state's vocabulary broke. It aborts the Send it
// happened in. A failure that belongs to the domain is NOT one of these — it is
// a message with an error in it, so that some state can decide what a failure
// means, which is the whole point of having states.
package statemachine
