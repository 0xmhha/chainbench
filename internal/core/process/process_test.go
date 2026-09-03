package process

import (
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// startSleeper launches a DETACHED long-lived process (a grandchild reparented to
// init) and returns its PID. Detaching mirrors how chainbench launches node
// processes — and, unlike a direct child, a detached process is reaped by init
// when killed, so it does not linger as a zombie that `kill -0` still sees.
func startSleeper(t *testing.T) int {
	t.Helper()
	// sh backgrounds the sleep and exits, so `sleep` is reparented to init; it
	// prints the sleep's PID. The sleep's fds are redirected to /dev/null so it
	// does not inherit the stdout pipe (which would keep .Output() blocked until
	// the sleep itself exits).
	out, err := exec.Command("sh", "-c", "sleep 300 >/dev/null 2>&1 & echo $!").Output()
	if err != nil {
		t.Fatalf("start sleeper: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		t.Fatalf("parse sleeper pid %q: %v", out, err)
	}
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
	return pid
}

// TestAlive pins the liveness probe the local driver's inspector uses: a
// running process reads alive, and one that has been killed (and reaped, since
// the sleeper is detached) reads gone.
func TestAlive(t *testing.T) {
	pid := startSleeper(t)
	if !Alive(pid) {
		t.Fatal("a started sleeper should be alive")
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	deadline := time.Now().Add(3 * time.Second)
	for Alive(pid) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if Alive(pid) {
		t.Fatal("a killed pid should not be alive")
	}
}

func TestAlive_LowPIDsAreNotAlive(t *testing.T) {
	for _, pid := range []int{0, 1, -1} {
		if Alive(pid) {
			t.Errorf("Alive(%d) = true, want false", pid)
		}
	}
}
