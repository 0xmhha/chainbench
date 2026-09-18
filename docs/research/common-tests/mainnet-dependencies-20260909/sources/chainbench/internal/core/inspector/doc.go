// Package inspector looks at a target and reports what is actually there.
//
// It answers three questions, on request only — nothing here watches:
//
//   - Ports:  which of these addresses already have a listener
//   - Paths:  which of these paths are missing or unusable on the target
//   - Hosts:  which of these hosts cannot be reached
//
// It reports facts and judges nothing. Deciding what a fact means — whether a
// busy port is a leftover of this workspace or someone else's, whether a
// missing genesis means "compose first" or "rebuild node 5" — belongs to the
// caller (chainsetup's start checks today, preflight's current-vs-target
// comparison in module-plan §M8).
//
// The alternative to asking first is finding out from the chain. A node whose
// port is taken dies with "address already in use" somewhere inside a launch
// sequence; a node whose binary is not on the server fails one step later
// with less to go on. Those are different situations with different remedies,
// and the inspector's job is to tell them apart before anything launches.
package inspector
