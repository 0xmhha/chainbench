package session_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func openComp(t *testing.T) session.Composition {
	t.Helper()
	c, err := session.OpenComposition(t.TempDir(), func() time.Time { return time.Unix(0, 0).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestLock_FreeThenHeldThenReleased is the ordinary cycle.
func TestLock_FreeThenHeldThenReleased(t *testing.T) {
	c := openComp(t)
	if _, state, err := c.Lock(); err != nil || state != session.LockFree {
		t.Fatalf("state = %s, err = %v; want free", state, err)
	}
	held, _, _, err := c.Acquire("net up --chain wemix")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	got, state, err := c.Lock()
	if err != nil || state != session.LockLive {
		t.Fatalf("state = %s, err = %v; want live", state, err)
	}
	if got.PID != os.Getpid() || got.Command != "net up --chain wemix" {
		t.Fatalf("lock = %+v; want this process and its command", got)
	}
	if err := held.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if _, state, _ := c.Lock(); state != session.LockFree {
		t.Fatalf("state = %s after release; want free", state)
	}
}

// TestLock_AnotherLiveRunIsRefused: the whole point is that two runs do not
// share a workspace without knowing. The refusal names who holds it, because
// "locked" on its own tells an operator nothing they can act on.
func TestLock_AnotherLiveRunIsRefused(t *testing.T) {
	c := openComp(t)
	other := exec.Command("/bin/sleep", "30")
	if err := other.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = other.Process.Kill(); _, _ = other.Process.Wait() })

	host, _ := os.Hostname()
	writeLock(t, c.Dir(), `{"pid":`+itoa(other.Process.Pid)+`,"host":"`+host+`","command":"first","at":"1970-01-01T00:00:00Z"}`)

	_, prev, state, err := c.Acquire("second")
	if !errors.Is(err, session.ErrLocked) {
		t.Fatalf("err = %v; want ErrLocked", err)
	}
	if state != session.LockLive || prev.Command != "first" {
		t.Fatalf("state = %s, prev = %+v; want the live first run", state, prev)
	}
}

// TestLock_TheSameRunMayTakeItTwice.
//
// A workspace is held by one run, not by one call: `net up` takes the lock and
// then calls the nine steps it is made of, each of which takes it too. Refusing
// there would deadlock the tool against itself, and releasing there would hand
// the workspace to a competitor while the run continued — which is what
// happened before this rule existed: the inner step's release removed the
// outer lock, and a concurrent `net allocate` walked straight in.
func TestLock_TheSameRunMayTakeItTwice(t *testing.T) {
	c := openComp(t)
	outer, _, _, err := c.Acquire("net up")
	if err != nil {
		t.Fatalf("outer Acquire: %v", err)
	}
	inner, _, state, err := c.Acquire("net allocate (inner step)")
	if err != nil {
		t.Fatalf("inner Acquire: %v — a run must not conflict with itself", err)
	}
	if state != session.LockLive {
		t.Fatalf("state = %s; the outer hold should still be live", state)
	}
	// Releasing the inner hold must not free the workspace.
	if err := inner.Release(); err != nil {
		t.Fatalf("inner Release: %v", err)
	}
	if _, state, _ := c.Lock(); state != session.LockLive {
		t.Fatalf("state = %s after the inner release; the run still holds it", state)
	}
	if got, _, _ := c.Lock(); got.Command != "net up" {
		t.Fatalf("lock command = %q; the outer holder must stay recorded", got.Command)
	}
	// The outermost release frees it.
	if err := outer.Release(); err != nil {
		t.Fatalf("outer Release: %v", err)
	}
	if _, state, _ := c.Lock(); state != session.LockFree {
		t.Fatalf("state = %s after the outer release; want free", state)
	}
}

// TestLock_ADeadRunsLockIsStaleAndTakenOver.
//
// This is the distinction the feature exists for: a workspace whose nodes are
// still running is either a test in progress or the wreck of one that died.
// A lock whose process is gone is the second, and it must not block the next
// run — but the previous holder is returned so the caller can say what was
// found rather than take over in silence.
func TestLock_ADeadRunsLockIsStaleAndTakenOver(t *testing.T) {
	c := openComp(t)
	// A real pid that is definitely gone: start a process and reap it.
	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	dead := cmd.Process.Pid

	host, _ := os.Hostname()
	writeLock(t, c.Dir(), `{"pid":`+itoa(dead)+`,"host":"`+host+`","command":"the run that died","at":"1970-01-01T00:00:00Z"}`)

	if _, state, _ := c.Lock(); state != session.LockStale {
		t.Fatalf("state = %s; want stale for a dead pid", state)
	}
	held, prev, state, err := c.Acquire("the next run")
	if err != nil {
		t.Fatalf("Acquire over a stale lock: %v", err)
	}
	if state != session.LockStale || prev.Command != "the run that died" {
		t.Fatalf("prev = %+v state = %s; the caller cannot report what it took over", prev, state)
	}
	_ = held.Release()
}

// TestLock_AnotherHostsLockIsNotJudged: a pid from another machine means
// nothing here, and calling it dead would be a confident wrong answer.
func TestLock_AnotherHostsLockIsNotJudged(t *testing.T) {
	c := openComp(t)
	writeLock(t, c.Dir(), `{"pid":1,"host":"some-other-box","command":"remote run","at":"1970-01-01T00:00:00Z"}`)
	if _, state, _ := c.Lock(); state != session.LockForeign {
		t.Fatalf("state = %s; want foreign", state)
	}
	if _, _, _, err := c.Acquire("mine"); !errors.Is(err, session.ErrLocked) {
		t.Fatalf("err = %v; a lock this host cannot judge must not be taken", err)
	}
}

func writeLock(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "workspace.lock"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
