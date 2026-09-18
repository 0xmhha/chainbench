//go:build e2e

package main

import (
	"regexp"
	"strconv"
	"syscall"
	"time"
)

// The e2e handoff tests drive the CLI and see its stdout, not its internals, so
// this is how they reach the nodes it started: the command prints each node's
// pid in its table, and a test that started a network is responsible for
// stopping it.
//
// It replaces a process.Manager these tests used to hold. That type was removed
// as dead code, correctly — nothing in production called it. What the removal
// could not see is that these files did, because a build tag hid them from the
// compiler. The lesson is in the CI job that now builds this tag, not in
// bringing the type back for two tests.

// pidLine matches the "pid=NNNN" the node table prints.
var pidLine = regexp.MustCompile(`pid=(\d+)`)

// pidsFrom reads the node pids out of a command's output.
func pidsFrom(out string) []int {
	var pids []int
	for _, m := range pidLine.FindAllStringSubmatch(out, -1) {
		if pid, err := strconv.Atoi(m[1]); err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	return pids
}

// stopPIDs asks each pid to exit, waits up to grace, then reports the ones
// still alive. A survivor is worth failing over: it holds ports the next test
// needs, and an orphan nobody names is how a suite starts failing for reasons
// that have nothing to do with the test that reports them.
func stopPIDs(pids []int, grace time.Duration) []int {
	for _, pid := range pids {
		_ = syscall.Kill(pid, syscall.SIGTERM)
	}
	deadline := time.Now().Add(grace)
	for {
		alive := livePIDs(pids)
		if len(alive) == 0 || time.Now().After(deadline) {
			if len(alive) > 0 {
				for _, pid := range alive {
					_ = syscall.Kill(pid, syscall.SIGKILL)
				}
				time.Sleep(200 * time.Millisecond)
				alive = livePIDs(pids)
			}
			return alive
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// livePIDs returns the pids that still exist. Signal 0 performs the existence
// check without delivering anything.
func livePIDs(pids []int) []int {
	var alive []int
	for _, pid := range pids {
		if syscall.Kill(pid, 0) == nil {
			alive = append(alive, pid)
		}
	}
	return alive
}
