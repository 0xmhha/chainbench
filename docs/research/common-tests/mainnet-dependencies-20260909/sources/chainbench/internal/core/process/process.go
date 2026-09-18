// Package process runs and records chain node processes for the setup pipeline.
// A Driver abstracts where/how a node runs (local, remote, attached) behind one
// surface; a Ledger persists which pid runs which binary under which command;
// LaunchAndRecord ties a launch to its ledger record; and StopNodeSet tears a
// launched node set down. Proc is the ledger's per-process record, and Alive is
// the liveness probe the local driver's inspector uses.
package process

import "syscall"

// Proc is one recorded process — what the run ledger persists per node.
type Proc struct {
	PID   int
	Label string
	// Binary is the executable's name or path, when the launcher knows it —
	// what lets the ledger answer "is a gstable already running there?".
	Binary string
	// Command is the launch command line, recorded for the operator reading
	// the ledger, never re-executed from here.
	Command string
	// DataDir is the process's data directory, if known. Removing it is a
	// separate operation from stopping the process (design S2).
	DataDir string
	// Host is the remote host the process runs on; empty for a local process
	// managed via OS signals.
	Host string
	// Revision counts how many times this label has been superseded in place —
	// 0 for a fresh launch, +1 on each binary swap (a hardfork). The prior
	// entry is archived in the ledger's history, so a swap preserves the pid
	// and command that ran before it rather than overwriting them.
	Revision int `json:"Revision,omitempty"`
}

// Alive reports whether pid is a live process (signal 0 probe).
func Alive(pid int) bool {
	if pid <= 1 {
		return false
	}
	// On unix, Kill with signal 0 performs error checking without sending a
	// signal: nil (alive) or EPERM (alive, not ours) => alive; ESRCH => gone.
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
