package process

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
)

// ProcessTable reads a machine's process table: whether a pid exists and the argv
// it runs.
type ProcessTable interface {
	PIDAlive(ctx context.Context, pid int) (bool, error)
	Cmdline(ctx context.Context, pid int) ([]string, error)
}

// Ownership is what a recorded pid is found to be.
type Ownership string

const (
	// OwnedRunning is the recorded launch, still running.
	OwnedRunning Ownership = "running"
	// OwnedMissing is a recorded pid that no longer exists.
	OwnedMissing Ownership = "missing"
	// OwnedMismatch is a live pid running something other than the recorded
	// launch: the pid was reused, and it is not this record's process.
	OwnedMismatch Ownership = "ownership_mismatch"
	// OwnedUnknown is a pid whose launch could not be established.
	OwnedUnknown Ownership = "unknown"
)

// LaunchVerdict is the verdict on one recorded pid, with why when it is not owned.
type LaunchVerdict struct {
	State  Ownership `json:"state"`
	PID    int       `json:"observedPid,omitempty"`
	Reason string    `json:"reason,omitempty"`
}

// ObserveLaunch reports whether pid is still the process launched as binary
// with args. A recorded pid is not proof of anything: only a live process whose
// argv is exactly that launch counts as running. Missing or unreadable argv
// establishes nothing, and the pid is read alive again after its argv so a
// process that exited in between is not taken for the record's.
func ObserveLaunch(ctx context.Context, o ProcessTable, pid int, binary string, args []string) LaunchVerdict {
	unknown := LaunchVerdict{State: OwnedUnknown, Reason: "probe_unavailable"}
	if pid <= 0 {
		return LaunchVerdict{State: OwnedUnknown, Reason: "no_recorded_pid"}
	}
	if o == nil || ctx.Err() != nil || binary == "" || len(args) == 0 {
		return unknown
	}
	alive, err := o.PIDAlive(ctx, pid)
	if err != nil {
		return unknown
	}
	missing := LaunchVerdict{State: OwnedMissing, Reason: "recorded_pid_absent"}
	if !alive {
		return missing
	}
	argv, err := o.Cmdline(ctx, pid)
	if err != nil || ctx.Err() != nil {
		return unknown
	}
	if len(argv) == 0 || filepath.Clean(argv[0]) != filepath.Clean(binary) || !slices.Equal(argv[1:], args) {
		return LaunchVerdict{State: OwnedMismatch, Reason: "argv_mismatch"}
	}
	if alive, err = o.PIDAlive(ctx, pid); err != nil || ctx.Err() != nil {
		return unknown
	}
	if !alive {
		return missing
	}
	return LaunchVerdict{State: OwnedRunning, PID: pid}
}

// ProcessTableOf is how the machine behind d is read. A darwin machine has no
// /proc, so its process table is read with ps; elsewhere the driver's own /proc
// reads answer. A driver that can do neither yields nil, which observes
// nothing.
func ProcessTableOf(ctx context.Context, d Driver) ProcessTable {
	if c, ok := d.(Commander); ok {
		if out, err := c.Run(ctx, "uname -s"); err == nil && strings.TrimSpace(out) == "Darwin" {
			return darwinObserver{c}
		}
	}
	o, _ := d.(ProcessTable)
	return o
}

// darwinObserver reads a darwin process table with ps, read-only. ps prints
// argv joined by spaces, so the executable column must agree with argv[0];
// an argument that contained a space cannot match a recorded one, which leaves
// such a process unowned rather than wrongly owned.
type darwinObserver struct{ c Commander }

func (o darwinObserver) PIDAlive(ctx context.Context, pid int) (bool, error) {
	if pid <= 0 {
		return false, errors.New("process: invalid pid")
	}
	out, err := o.c.Run(ctx, fmt.Sprintf("if ! command -v ps >/dev/null 2>&1; then printf unavailable; elif ps -p %d -o pid= >/dev/null 2>&1; then printf alive; else printf absent; fi", pid))
	if err != nil {
		return false, err
	}
	switch strings.TrimSpace(out) {
	case "alive":
		return true, nil
	case "absent":
		return false, nil
	}
	return false, errors.New("process: process inspection unavailable")
}

func (o darwinObserver) Cmdline(ctx context.Context, pid int) ([]string, error) {
	if pid <= 0 {
		return nil, errors.New("process: invalid pid")
	}
	raw, err := o.c.Run(ctx, fmt.Sprintf("ps -ww -p %d -o args=", pid))
	if err != nil {
		return nil, err
	}
	executable, err := o.c.Run(ctx, fmt.Sprintf("ps -ww -p %d -o comm=", pid))
	if err != nil {
		return nil, err
	}
	argv := strings.Fields(raw)
	if len(argv) == 0 || filepath.Clean(argv[0]) != filepath.Clean(strings.TrimSpace(executable)) {
		return nil, errors.New("process: launch identity unavailable")
	}
	return argv, nil
}
