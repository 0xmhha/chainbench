package statemachine

// What says which message this is.
//
// It is an integer rather than a string because the bands it is cut into carry
// meaning: a machine gets a base, and within that base the value says whether a
// message comes from outside, goes up to a controller, or is the machine
// talking to itself. See protocol.go for the bases, and each machine's own
// protocol.go for the messages.
type What int

// Message is what crosses a machine's queue.
//
// The concrete type is what a state switches on, because that is where the
// message's arguments are. What is for the log and for the band checks — it
// answers "which message" without the reader having to know the type.
type Message interface{ What() What }
