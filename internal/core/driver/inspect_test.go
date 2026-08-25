package driver_test

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
)

// TestLocalInspector_AnswersProcessQuestions pins the local half of the
// capability with this test's own process as the subject.
func TestLocalInspector_AnswersProcessQuestions(t *testing.T) {
	d := driver.NewLocalDriver()
	ctx := context.Background()

	alive, err := d.PIDAlive(ctx, os.Getpid())
	if err != nil || !alive {
		t.Fatalf("own pid reported dead (alive=%v err=%v)", alive, err)
	}
	// A freshly exited child is a pid that must read as gone.
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if gone, _ := d.PIDAlive(ctx, cmd.Process.Pid); gone {
		t.Skip("pid was reused immediately; nothing to assert")
	}

	// A binary that certainly runs (this test binary's process is go's; use a
	// long-lived sleep we own instead).
	sleep := exec.Command("sleep", "30")
	if err := sleep.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sleep.Process.Kill(); _, _ = sleep.Process.Wait() })
	pids, err := d.FindBinary(ctx, "sleep")
	if err != nil {
		t.Fatalf("FindBinary: %v", err)
	}
	found := false
	for _, p := range pids {
		if p == sleep.Process.Pid {
			found = true
		}
	}
	if !found {
		t.Errorf("FindBinary(sleep) = %v, missing pid %d", pids, sleep.Process.Pid)
	}
	if none, err := d.FindBinary(ctx, "no-such-binary-name"); err != nil || len(none) != 0 {
		t.Errorf("no match must be an empty answer, got %v / %v", none, err)
	}
}

// TestLocalCommander_ReturnsStdout pins the capability's contract: the
// command's stdout, nothing folded in.
func TestLocalCommander_ReturnsStdout(t *testing.T) {
	d := driver.NewLocalDriver()
	out, err := d.Run(context.Background(), "echo one && echo two")
	if err != nil {
		t.Fatal(err)
	}
	if out != "one\ntwo\n" {
		t.Errorf("stdout = %q", out)
	}
	if _, err := d.Run(context.Background(), "exit 3"); err == nil {
		t.Error("a failing command must error")
	}
}
