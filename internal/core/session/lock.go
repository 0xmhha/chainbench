package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// lockFile is the name a workspace's lock takes inside its control directory.
const lockFile = "workspace.lock"

// Lock is the record a run leaves while it holds a workspace: who is using it,
// from where, since when, and doing what.
//
// It exists to answer one question an operator cannot otherwise answer — a
// network is up, so is a test running right now, or did an earlier run die and
// leave it? The pid answers it, but only together with the host: a pid from
// another machine says nothing here, and treating it as dead would be a
// confident wrong answer.
type Lock struct {
	PID     int    `json:"pid"`
	Host    string `json:"host"`
	Command string `json:"command"`
	At      string `json:"at"`
}

// LockState is what a workspace's lock says about the run that took it.
type LockState string

const (
	// LockFree means no run holds the workspace.
	LockFree LockState = "free"
	// LockLive means the recorded process is still running on this host: a
	// test is in progress and the nodes belong to it.
	LockLive LockState = "live"
	// LockStale means the recorded process is gone. Whatever it started was
	// left behind — this is the leftover case, and it is safe to take over.
	LockStale LockState = "stale"
	// LockForeign means the lock was taken on another host, so this machine
	// cannot judge whether it is alive. Not free, not provably stale.
	LockForeign LockState = "foreign"
)

// ErrLocked is returned when a workspace is held by a live run.
var ErrLocked = errors.New("session: workspace is in use")

// Held is an acquired lock. Release removes it; a run that dies without
// releasing leaves it behind, which is exactly the evidence the next run needs.
type Held struct {
	path string
	lock Lock
	// nested marks a re-entrant take: this run already held the workspace, so
	// releasing here would hand it to a competitor while the run continues.
	// Only the outermost hold releases.
	nested bool
}

// Lock returns the workspace's lock and what it means, without taking it.
func (c Composition) Lock() (Lock, LockState, error) {
	return readLock(filepath.Join(c.dir, lockFile))
}

// Acquire takes the workspace's lock for this process.
//
// A stale lock is taken over rather than refused: the run that left it is gone,
// and refusing would make an operator delete a file to get moving, which is a
// habit that also deletes live ones. What it must not do is take over silently
// — the previous holder is returned so the caller can say what it found.
func (c Composition) Acquire(command string) (*Held, Lock, LockState, error) {
	path := filepath.Join(c.dir, lockFile)
	prev, state, err := readLock(path)
	if err != nil {
		return nil, Lock{}, "", err
	}
	switch state {
	case LockLive:
		// A workspace is held by one run, not by one call. A composite step
		// (net up) takes the lock and then calls the steps it is made of, and
		// each of those takes it too; refusing there would make the tool
		// deadlock against itself, and releasing there would hand the
		// workspace to a competitor mid-run. Another process is the real
		// conflict.
		if prev.PID == os.Getpid() {
			return &Held{path: path, lock: prev, nested: true}, prev, state, nil
		}
		return nil, prev, state, fmt.Errorf("%w: %s", ErrLocked, prev.Describe())
	case LockForeign:
		return nil, prev, state, fmt.Errorf("%w: %s", ErrLocked, prev.Describe())
	}

	host, _ := os.Hostname()
	mine := Lock{PID: os.Getpid(), Host: host, Command: command, At: c.now().UTC().Format(time.RFC3339)}
	b, err := json.MarshalIndent(mine, "", "  ")
	if err != nil {
		return nil, prev, state, fmt.Errorf("session: marshal lock: %w", err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return nil, prev, state, fmt.Errorf("session: write %s: %w", lockFile, err)
	}
	return &Held{path: path, lock: mine}, prev, state, nil
}

// Release removes the lock. Releasing a lock another run has since taken is not
// an error worth failing a command over, but it is not silent either: the file
// is only removed when it still names this process.
func (h *Held) Release() error {
	if h == nil || h.nested {
		return nil
	}
	cur, _, err := readLock(h.path)
	if err != nil || cur.PID != h.lock.PID {
		return nil
	}
	if err := os.Remove(h.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("session: remove %s: %w", lockFile, err)
	}
	return nil
}

// readLock reads the lock file and classifies it.
func readLock(path string) (Lock, LockState, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Lock{}, LockFree, nil
	}
	if err != nil {
		return Lock{}, "", fmt.Errorf("session: read %s: %w", lockFile, err)
	}
	var l Lock
	if err := json.Unmarshal(b, &l); err != nil {
		// An unreadable lock is not a live one, and refusing on it would strand
		// the workspace. It is reported as stale so the caller says what it saw.
		return Lock{}, LockStale, nil
	}
	host, _ := os.Hostname()
	if l.Host != "" && !strings.EqualFold(l.Host, host) {
		return l, LockForeign, nil
	}
	if l.PID <= 0 || !processAlive(l.PID) {
		return l, LockStale, nil
	}
	return l, LockLive, nil
}

// processAlive reports whether a pid is a live process on this host. Signal 0
// performs the permission and existence checks without delivering anything.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}

// Describe renders a lock for a message an operator can act on.
func (l Lock) Describe() string {
	cmd := l.Command
	if cmd == "" {
		cmd = "(command not recorded)"
	}
	return fmt.Sprintf("pid %d on %s since %s: %s", l.PID, l.Host, l.At, cmd)
}
