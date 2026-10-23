// Package process owns how a node comes up and goes down.
//
// It holds the primitives a launch is made of: a Driver that provisions files,
// inits a datadir and starts or stops a process on one machine (LocalDriver
// and RemoteDriver), NodeConfig to assemble a node's launch spec,
// LaunchAndRecord to start a node and write its pid to the Ledger, and
// StopNodeSet to bring a set down. The order of a bring-up (phases, actions,
// retries) is not here: chainsetup's start step owns it. The launch policy
// that once lived here (Direct and Launcher) had no callers left and was
// removed.
//
// The driver executes and this package decides; the process package owns the
// pid ledger and the stop policy (SIGTERM, grace, SIGKILL, verify).
//
// The word "supervisor" is deliberately gone from the code: it reads as the
// privileged-operator role (sudo), which lives with the resource module's
// server set and the driver's elevated runner, not here.
package process
