package procman

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

func TestStopAll_TerminatesAndVerifies(t *testing.T) {
	m := New()
	p1 := startSleeper(t)
	p2 := startSleeper(t)
	m.Track(p1, "a")
	m.Track(p2, "b")
	if m.Count() != 2 {
		t.Fatalf("Count = %d, want 2", m.Count())
	}
	if !Alive(p1) || !Alive(p2) {
		t.Fatal("sleepers should be alive before StopAll")
	}

	if leaks := m.StopAll(3 * time.Second); len(leaks) != 0 {
		t.Fatalf("StopAll leaked %v, want none", leaks)
	}
	if Alive(p1) || Alive(p2) {
		t.Fatal("sleepers still alive after StopAll")
	}
}

func TestTrack_DedupAndIgnoresLowPIDs(t *testing.T) {
	m := New()
	m.Track(42, "x")
	m.Track(42, "x-again") // dup
	m.Track(0, "attached") // ignored
	m.Track(1, "init")     // ignored
	if m.Count() != 1 {
		t.Fatalf("Count = %d, want 1 (dedup + low-pid ignore)", m.Count())
	}
}

func TestTrackFromOutput(t *testing.T) {
	out := "" +
		"handoff wemix -> wbft; launching...\n" +
		"  node1  http://127.0.0.1:40010  pid=1234\n" +
		"  node2  http://127.0.0.1:40020  pid=5678\n" +
		"governance deployed\n"
	m := New()
	if n := m.TrackFromOutput(out); n != 2 {
		t.Fatalf("tracked %d, want 2", n)
	}
	got := map[int]bool{}
	for _, p := range m.Tracked() {
		got[p.PID] = true
	}
	if !got[1234] || !got[5678] {
		t.Fatalf("tracked PIDs = %v, want 1234 and 5678", m.Tracked())
	}
}

func TestStopAll_Idempotent(t *testing.T) {
	m := New()
	p := startSleeper(t)
	m.Track(p, "a")
	if leaks := m.StopAll(3 * time.Second); len(leaks) != 0 {
		t.Fatalf("first StopAll leaked %v", leaks)
	}
	// A second call over already-dead PIDs must be a clean no-op.
	if leaks := m.StopAll(time.Second); len(leaks) != 0 {
		t.Fatalf("second StopAll leaked %v", leaks)
	}
}

func TestAlive_GonePID(t *testing.T) {
	p := startSleeper(t)
	m := New()
	m.Track(p, "a")
	m.StopAll(3 * time.Second)
	if Alive(p) {
		t.Fatal("Alive should be false for a killed PID")
	}
}

func TestStopOne_TerminatesJustThatProcess(t *testing.T) {
	m := New()
	keep := startSleeper(t)
	target := startSleeper(t)
	m.Track(keep, "keep")
	m.Track(target, "target")

	if err := m.StopOne(target, time.Second); err != nil {
		t.Fatalf("StopOne: %v", err)
	}
	if Alive(target) {
		t.Fatal("target still alive after StopOne")
	}
	if !Alive(keep) {
		t.Fatal("StopOne stopped an untargeted process")
	}
	_ = m.StopAll(time.Second)
}

func TestStopOne_UntrackedPIDIsAnError(t *testing.T) {
	m := New()
	if err := m.StopOne(999999, time.Second); err == nil {
		t.Fatal("expected an error for an untracked pid")
	}
}

func TestStopOne_AlreadyGoneIsNotAnError(t *testing.T) {
	m := New()
	pid := startSleeper(t)
	m.Track(pid, "one")
	if err := m.StopOne(pid, time.Second); err != nil {
		t.Fatalf("first StopOne: %v", err)
	}
	if err := m.StopOne(pid, time.Second); err != nil {
		t.Fatalf("second StopOne on a stopped process: %v", err)
	}
}
