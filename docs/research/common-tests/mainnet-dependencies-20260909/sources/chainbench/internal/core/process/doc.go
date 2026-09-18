// Package launcher owns how a node comes up and goes down.
//
// Two halves, one module. Direct is how to launch: arm a plan into node specs
// (argv, config, identity), materialize the files on the target, init each
// datadir, start the processes. Launcher is how to keep launching until the
// network is up: phase order, the health gate, backoff and retry, teardown of
// what did not come up, and a diagnosis that says why. They were two packages
// (chainsetup.LocalLauncher and core/supervisor) that were the top and bottom
// half of one job.
//
// The driver executes and this package decides; the process package owns the
// pid ledger and the stop policy (SIGTERM, grace, SIGKILL, verify). Three
// entry points, three different jobs — the eight that existed before were
// wrappers of these three with a copy of the node-assembly in each.
//
// The word "supervisor" is deliberately gone from the code: it reads as the
// privileged-operator role (sudo), which lives with the resource module's
// server set and the driver's elevated runner, not here.
package process
