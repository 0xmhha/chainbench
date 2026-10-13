package session

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// lockChildEnv makes this test binary act as one competing process.
const lockChildEnv = "CHAINBENCH_LOCK_RACE_CHILD"

// TestAcquireLock_TwoProcessesNeverBothHoldIt: two processes that find the
// workspace free at the same moment must not both take it. The window between
// reading the lock and writing it is widened so the race happens every time.
func TestAcquireLock_TwoProcessesNeverBothHoldIt(t *testing.T) {
	if path := os.Getenv(lockChildEnv); path != "" {
		lockRaceChild(path)
		return
	}
	dir := t.TempDir()
	path := filepath.Join(dir, lockFile)
	start := filepath.Join(dir, "start")
	children := make([]*exec.Cmd, 2)
	outs := make([]*strings.Builder, 2)
	for i := range children {
		outs[i] = &strings.Builder{}
		c := exec.Command(os.Args[0], "-test.run=^TestAcquireLock_TwoProcessesNeverBothHoldIt$")
		c.Env = append(os.Environ(), lockChildEnv+"="+path)
		c.Stdout, c.Stderr = outs[i], outs[i]
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		children[i] = c
	}
	time.Sleep(300 * time.Millisecond)
	if err := os.WriteFile(start, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	held := 0
	for i, c := range children {
		_ = c.Wait()
		if strings.Contains(outs[i].String(), "HELD") {
			held++
		}
	}
	if held != 1 {
		t.Fatalf("%d processes held the lock at once, want exactly 1\nchild 1: %s\nchild 2: %s", held, outs[0], outs[1])
	}
}

// lockRaceChild waits for the start signal, takes the lock with a widened
// read-to-write window, and keeps it long enough for the other child to try.
func lockRaceChild(path string) {
	start := filepath.Join(filepath.Dir(path), "start")
	for {
		if _, err := os.Stat(start); err == nil {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	afterLockRead = func() { time.Sleep(400 * time.Millisecond) }
	h, _, _, err := AcquireLock(path, "race", nil)
	if err != nil {
		fmt.Println("REFUSED", err)
		return
	}
	fmt.Println("HELD")
	time.Sleep(time.Second)
	_ = h.Release()
}
