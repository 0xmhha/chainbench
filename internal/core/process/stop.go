package process

import (
	"context"
	"syscall"
	"time"
)

// How a node is asked to go down.
//
// A chain node keeps a database. Killed outright it has no chance to close it,
// and the node that comes back finds an unclean shutdown, switches to snap sync
// because its local state is incomplete, and discards the chain it had. If the
// rest of the network is meanwhile halted — which is exactly the situation a
// restart is meant to end — nothing can serve that sync and the node never
// rejoins. Measured 2026-09-05: two runs in five of the wbft quorum-recovery
// scenario, with `Unclean shutdown detected` and `Switch sync mode from full
// sync to snap sync: snap sync incomplete` in the restarted node's log.
//
// So a stop asks first and insists after: SIGTERM, a grace period in which the
// node may close its database, and SIGKILL only for a node that would otherwise
// never go. The cost is that stopping takes as long as the node needs, up to
// the grace; the alternative is a node that stops quickly and cannot come back.
const (
	// StopGrace is how long a node has to shut down of its own accord.
	//
	// Generous, because the expensive part is flushing a database and the
	// penalty for cutting it short is losing the chain rather than waiting.
	StopGrace = 15 * time.Second
	// stopPoll is how often the process is checked for having gone.
	stopPoll = 100 * time.Millisecond
	// killWait is how long SIGKILL is given to take effect. A signal is
	// delivered rather than instant, so a stop that returned the moment it was
	// sent could report a node gone that a caller then finds running.
	killWait = 5 * time.Second
)

// awaitExit waits for a pid to disappear, up to grace. It reports whether the
// process went on its own.
//
// It reaps as it polls, because a process this one launched becomes a zombie
// when it exits and stays in the process table until someone collects it —
// and a zombie answers kill(pid, 0) exactly like a running process. Without
// the reap, a node that shut down promptly would look alive for the whole
// grace and then be sent a SIGKILL it had no need of, which is the behaviour
// this policy exists to avoid.
func awaitExit(ctx context.Context, pid int, grace time.Duration) bool {
	deadline := time.Now().Add(grace)
	for {
		if gone(pid) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		select {
		case <-ctx.Done():
			return gone(pid)
		case <-time.After(stopPoll):
		}
	}
}

// gone reports whether a pid has left the process table, collecting it first if
// it is this process's child and has already exited.
//
// The reap is best-effort: wait4 fails with ECHILD for a pid that is not ours,
// which is the normal case — a node is usually launched by one chainbench
// invocation and stopped by another — and there is nothing to collect then.
func gone(pid int) bool {
	var status syscall.WaitStatus
	if reaped, err := syscall.Wait4(pid, &status, syscall.WNOHANG, nil); err == nil && reaped == pid {
		return true
	}
	return !Alive(pid)
}
