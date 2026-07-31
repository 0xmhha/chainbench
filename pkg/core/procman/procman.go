// Package procman is a lifecycle manager for locally-launched node processes. It
// tracks the PIDs chainbench starts (parsed from a launch command's output or a
// nodeset), and guarantees they are actually gone on teardown: SIGTERM, then a
// SIGKILL escalation, then a verification pass that reports any survivors as
// leaks. It exists so test harnesses (and the local driver) stop relying on
// best-effort `pkill -f <datadir>` sweeps, which leave orphans that hold ports
// (e.g. the go-wemix producer's embedded etcd) and make subsequent runs flaky.
package procman

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// Proc is one managed process.
type Proc struct {
	PID   int
	Label string
}

// Manager tracks a set of launched processes and tears them down verifiably.
type Manager struct {
	mu    sync.Mutex
	procs []Proc
	seen  map[int]bool
}

// New returns an empty Manager.
func New() *Manager { return &Manager{seen: map[int]bool{}} }

// Track registers a PID (deduplicated). PIDs <= 1 are ignored (0 = attached /
// unknown; 1 = init, never ours).
func (m *Manager) Track(pid int, label string) {
	if pid <= 1 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.seen[pid] {
		return
	}
	m.seen[pid] = true
	m.procs = append(m.procs, Proc{PID: pid, Label: label})
}

var pidRe = regexp.MustCompile(`pid=(\d+)`)

// TrackFromOutput parses `pid=<n>` occurrences from a launch command's output
// (e.g. `chainbench upgrade run` prints `node1  http://...  pid=1234`) and tracks
// each. Returns the number newly tracked.
func (m *Manager) TrackFromOutput(out string) int {
	before := m.Count()
	for _, match := range pidRe.FindAllStringSubmatch(out, -1) {
		if pid, err := strconv.Atoi(match[1]); err == nil {
			m.Track(pid, "launch-output")
		}
	}
	return m.Count() - before
}

// TrackNodeSet reads <dataDir>/nodeset.json and tracks each node's pid. Missing
// or unreadable files are not an error (the launch may have failed before
// writing it); it just tracks nothing.
func (m *Manager) TrackNodeSet(dataDir string) int {
	b, err := os.ReadFile(filepath.Join(dataDir, "nodeset.json"))
	if err != nil {
		return 0
	}
	var ns struct {
		Nodes []struct {
			Index int `json:"index"`
			PID   int `json:"pid"`
		} `json:"nodes"`
	}
	if json.Unmarshal(b, &ns) != nil {
		return 0
	}
	before := m.Count()
	for _, n := range ns.Nodes {
		m.Track(n.PID, "node"+strconv.Itoa(n.Index))
	}
	return m.Count() - before
}

// Count returns the number of tracked processes.
func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.procs)
}

// Tracked returns a copy of the tracked processes.
func (m *Manager) Tracked() []Proc {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Proc, len(m.procs))
	copy(out, m.procs)
	return out
}

// Alive reports whether pid is a live process (signal 0 probe).
func Alive(pid int) bool {
	if pid <= 1 {
		return false
	}
	// On unix, Kill with signal 0 performs error checking without sending a
	// signal: nil (alive) or EPERM (alive, not ours) => alive; ESRCH => gone.
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// StopAll terminates every tracked process and verifies it is gone: it sends
// SIGTERM, waits up to grace for a clean exit, SIGKILLs any survivor, then polls
// briefly and returns the PIDs still alive (leaks — expected to be empty). It is
// safe to call more than once.
func (m *Manager) StopAll(grace time.Duration) []int {
	procs := m.Tracked()

	// Phase 1: polite SIGTERM to all still-alive.
	for _, p := range procs {
		if Alive(p.PID) {
			_ = syscall.Kill(p.PID, syscall.SIGTERM)
		}
	}
	if waitAllGone(procs, grace) {
		return nil
	}

	// Phase 2: SIGKILL survivors.
	for _, p := range procs {
		if Alive(p.PID) {
			_ = syscall.Kill(p.PID, syscall.SIGKILL)
		}
	}
	waitAllGone(procs, 5*time.Second)

	// Phase 3: report leaks.
	var leaks []int
	for _, p := range procs {
		if Alive(p.PID) {
			leaks = append(leaks, p.PID)
		}
	}
	return leaks
}

// waitAllGone polls until every proc is gone or timeout elapses; returns whether
// all are gone.
func waitAllGone(procs []Proc, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		allGone := true
		for _, p := range procs {
			if Alive(p.PID) {
				allGone = false
				break
			}
		}
		if allGone {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}
