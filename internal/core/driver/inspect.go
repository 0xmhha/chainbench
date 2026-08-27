package driver

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/process"
)

// ProcessInspector is an optional Driver capability: ask the machine the
// process questions a pre-launch check needs — whether a pid is alive, and
// which pids run a named binary. Like PortProber, it answers ON the machine:
// an operator-side guess about a remote process is a guess about the wrong
// machine.
type ProcessInspector interface {
	// PIDAlive reports whether the pid exists on the machine.
	PIDAlive(ctx context.Context, pid int) (bool, error)
	// FindBinary returns the pids running the exactly-named binary.
	FindBinary(ctx context.Context, name string) ([]int, error)
}

// Commander is an optional Driver capability: run one shell command on the
// machine and return its stdout. It exists for the checks and small reads a
// bring-up needs between the structured steps; anything that launches a node
// stays on Launch, where the ledger records it.
type Commander interface {
	Run(ctx context.Context, command string) (string, error)
}

// PIDAlive answers with a signal-0 probe, the same check the process
// package's teardown verification uses.
func (d *LocalDriver) PIDAlive(_ context.Context, pid int) (bool, error) {
	return process.Alive(pid), nil
}

// FindBinary asks pgrep for exact-name matches. No match is an empty answer,
// not an error — that is the answer the caller wants.
func (d *LocalDriver) FindBinary(ctx context.Context, name string) ([]int, error) {
	out, err := exec.CommandContext(ctx, "pgrep", "-x", name).Output()
	if err != nil {
		// pgrep exits 1 for "no processes matched".
		if ee, ok := err.(*exec.ExitError); ok && ee.ExitCode() == 1 {
			return nil, nil
		}
		return nil, fmt.Errorf("driver: pgrep %s: %w", name, err)
	}
	return parsePids(string(out))
}

// Run executes command through the shell and returns its stdout.
func (d *LocalDriver) Run(ctx context.Context, command string) (string, error) {
	out, err := exec.CommandContext(ctx, "/bin/sh", "-c", command).Output()
	if err != nil {
		return "", fmt.Errorf("driver: run %q: %w", command, err)
	}
	return string(out), nil
}

// PIDAlive asks the machine's /proc, which answers for every process
// regardless of ownership — kill -0 reads EPERM on another user's process as
// absence, and the servers this targets log in as an unprivileged user.
func (d *RemoteDriver) PIDAlive(ctx context.Context, pid int) (bool, error) {
	out, err := d.sh(ctx, fmt.Sprintf("test -d /proc/%d && echo yes || echo no", pid))
	if err != nil {
		return false, fmt.Errorf("driver: remote pid probe: %w", err)
	}
	return strings.TrimSpace(out) == "yes", nil
}

// FindBinary asks the machine's pgrep for exact-name matches.
func (d *RemoteDriver) FindBinary(ctx context.Context, name string) ([]int, error) {
	out, err := d.sh(ctx, "pgrep -x "+shq(name)+" || true")
	if err != nil {
		return nil, fmt.Errorf("driver: remote pgrep %s: %w", name, err)
	}
	return parsePids(out)
}

// Run executes command on the machine and returns its stdout.
func (d *RemoteDriver) Run(ctx context.Context, command string) (string, error) {
	out, err := d.sh(ctx, command)
	if err != nil {
		return "", fmt.Errorf("driver: remote run: %w", err)
	}
	return out, nil
}

func parsePids(out string) ([]int, error) {
	var pids []int
	for _, f := range strings.Fields(out) {
		p, err := strconv.Atoi(f)
		if err != nil {
			return nil, fmt.Errorf("driver: unexpected pgrep output %q", f)
		}
		pids = append(pids, p)
	}
	return pids, nil
}
