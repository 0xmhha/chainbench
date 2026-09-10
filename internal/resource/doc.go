// Package resource owns what a network is composed from: which servers exist,
// how to reach them, which port slots they offer, and which of those are
// already taken.
//
// It is the single owner of a question that used to be answered in six places.
// The placement types, the port arithmetic, the server inventory, the attached
// address map and the role vocabulary lived in core/netmap, netmap,
// netmap/internal/serverset, core/portplan, core/place and core/netreg; the
// consolidation folded all of them here (and the attached-network registry into
// core/session). The file the operator keeps is one server set, and this package
// is the only thing that reads it.
//
// What it holds:
//
//	serverset.go       the server set: hosts, port bands, credentials
//	workspaceconfig.go the environment file: the target dataRoot and its
//	                   purpose directories, so one DSL runs across targets
//	inventory.go       what is already allocated, derived from the workspaces
//	pool.go, ports.go  the slots a set offers and the arithmetic behind them
//	placement.go       assigning nodes to servers and slots
//	localmap.go        the --docker address substitution
//	opener.go          the one place a path is turned into a reachable location
//	machine.go         commands issued to a server directly
//	baseline.go        the approved fingerprint of an environment's inputs
//
// The addresses in the server set are the REAL ones, always. Nodes must reach
// each other by them — they go into the genesis and the static-nodes list — so
// substituting a loopback address there would poison the artifacts. localmap
// applies only when this tool dials a server itself, and only under --docker.
// The file can sit there unused; the flag is the switch.
package resource
