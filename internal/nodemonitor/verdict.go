package nodemonitor

// Verdict is one node's readiness classification. The order is the escalation
// order the gate acts on: a set of nodes is only READY when every node is, and
// the worst verdict present drives the gate's next action.
type Verdict int

const (
	// Ready means the node is fit to run a test on: process alive, RPC ready,
	// block height advancing, not syncing, peers and consensus participation as
	// the topology wants, on the intended chain, not forked.
	Ready Verdict = iota
	// Waitable means the node is alive and reachable but not there yet — still
	// syncing, not yet advancing, short of its peer target, or not yet
	// participating in consensus. Waiting may clear it.
	Waitable
	// Restartable means a restart may clear it: the process is not alive, or its
	// RPC never came up, or a launch failure that a fresh start can resolve.
	Restartable
	// Fatal means clearing it would need a destructive remedy (deleting data,
	// rewinding, swapping genesis) or the node is on the wrong chain or has
	// diverged. The gate never applies such a remedy automatically; it
	// terminates so an operator decides.
	Fatal
)

// String returns the verdict label.
func (v Verdict) String() string {
	switch v {
	case Ready:
		return "READY"
	case Waitable:
		return "WAITABLE"
	case Restartable:
		return "RESTARTABLE"
	case Fatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
