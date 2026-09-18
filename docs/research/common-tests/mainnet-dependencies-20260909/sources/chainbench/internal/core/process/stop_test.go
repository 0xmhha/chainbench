package process_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/process"
)

// A chain node keeps a database, so how it is asked to stop decides whether it
// can come back. These drive real processes, because the thing under test is
// which signal arrives and what the process gets to do about it — a fake would
// only restate the code.

// sleeper starts a process that ignores nothing and exits on SIGTERM, and
// returns its pid.
func sleeper(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("sh", "-c", "sleep 300")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	return cmd.Process.Pid
}

// stubborn starts a process that traps SIGTERM and keeps running, so the
// escalation can be observed.
//
// It reports readiness by creating a file, and this waits for it. Signalling
// before the shell has run its trap kills the process outright, which looks
// exactly like the escalation working and would have made the test pass for
// the wrong reason — it did, until the delay was added.
func stubborn(t *testing.T) int {
	t.Helper()
	ready := filepath.Join(t.TempDir(), "ready")
	// The loop matters too: `trap '' TERM; sleep 300` lets the shell replace
	// itself with sleep, and the process that ignores the signal is then gone.
	cmd := exec.Command("sh", "-c",
		"trap '' TERM; : > "+ready+"; while true; do sleep 1; done")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })

	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(ready); err == nil {
			return cmd.Process.Pid
		}
		if time.Now().After(deadline) {
			t.Fatal("the stubborn fixture never signalled that its trap was installed")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// TestStop_AsksBeforeItInsists: a node that will go on SIGTERM is not killed.
//
// This is the whole point of the policy. A killed node has no chance to close
// its database, and the node that comes back finds an unclean shutdown. What
// proves the signal was SIGTERM is that the process is gone without SIGKILL
// having been needed — measured here by the stop returning well inside the
// grace period, which a SIGKILL-only stop would also do, so the companion test
// below covers the other half.
func TestStop_AsksBeforeItInsists(t *testing.T) {
	pid := sleeper(t)
	d := process.NewLocalDriver()

	start := time.Now()
	if err := d.Stop(context.Background(), process.Handle{Index: 1, PID: pid}); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if took := time.Since(start); took > process.StopGrace {
		t.Errorf("stopping a cooperative process took %s, longer than the grace of %s", took, process.StopGrace)
	}
	if process.Alive(pid) {
		t.Error("the process is still running after Stop returned")
	}
}

// TestStop_InsistsOnAProcessThatWillNotGo: a node that ignores SIGTERM is still
// stopped, because one that keeps running holds the ports and datadir of the
// node the caller asked to stop.
func TestStop_InsistsOnAProcessThatWillNotGo(t *testing.T) {
	if testing.Short() {
		t.Skip("waits out the shutdown grace")
	}
	pid := stubborn(t)
	d := process.NewLocalDriver()

	start := time.Now()
	if err := d.Stop(context.Background(), process.Handle{Index: 1, PID: pid}); err != nil {
		t.Fatalf("stop: %v", err)
	}
	// It had to wait: a stop that returned immediately would mean SIGTERM was
	// never given a chance, which is the behaviour this policy replaced.
	if took := time.Since(start); took < process.StopGrace {
		t.Errorf("a process that ignores SIGTERM was killed after %s, before the grace of %s elapsed",
			took, process.StopGrace)
	}
	if process.Alive(pid) {
		t.Error("a process that ignores SIGTERM was left running")
	}
}

// TestStop_AProcessAlreadyGoneIsNotAnError: teardown runs over a recorded PID
// list, and a node that already exited is the normal case rather than a fault.
func TestStop_AProcessAlreadyGoneIsNotAnError(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid

	d := process.NewLocalDriver()
	if err := d.Stop(context.Background(), process.Handle{Index: 1, PID: pid}); err != nil {
		t.Errorf("stopping an already-exited process reported an error: %v", err)
	}
}

// TestStop_HonoursACancelledContext: teardown under a cancelled context must
// not sit out the full grace for every node.
func TestStop_HonoursACancelledContext(t *testing.T) {
	pid := stubborn(t)
	d := process.NewLocalDriver()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	_ = d.Stop(ctx, process.Handle{Index: 1, PID: pid})
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("a cancelled stop still waited %s", took)
	}
	// And it still stopped the process: a cancelled context shortens the wait,
	// it does not leave a node running.
	if process.Alive(pid) {
		t.Error("a cancelled stop left the process running")
	}
}
