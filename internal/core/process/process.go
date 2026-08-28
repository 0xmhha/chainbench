// Package procman is a lifecycle manager for locally-launched node processes. It
// tracks the PIDs chainbench starts (parsed from a launch command's output or a
// nodeset), and guarantees they are actually gone on teardown: SIGTERM, then a
// SIGKILL escalation, then a verification pass that reports any survivors as
// leaks. It exists so test harnesses (and the local driver) stop relying on
// best-effort `pkill -f <datadir>` sweeps, which leave orphans that hold ports
// (e.g. the go-wemix producer's embedded etcd) and make subsequent runs flaky.
//
// It also tracks each process's data directory and remote host: StopAll stops
// and verifies local processes, StopRemote stops remote ones via an injected
// kill, and RemoveDataDirs deletes data directories as a step separate from
// stopping (design S2).
package process

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Proc is one managed process.
type Proc struct {
	PID   int
	Label string
	// Binary is the executable's name or path, when the launcher knows it —
	// what lets the ledger answer "is a gstable already running there?".
	Binary string
	// Command is the launch command line, recorded for the operator reading
	// the ledger, never re-executed from here.
	Command string
	// DataDir is the process's data directory, if known. Removing it is a
	// separate operation from stopping the process (design S2).
	DataDir string
	// Host is the remote host the process runs on; empty for a local process
	// managed via OS signals.
	Host string
}

// localHosts are the host spellings that still mean "this machine". The local
// launcher records each node's host, and for a local network that is a loopback
// address — which must not read as remote, or the process would be excluded from
// every signal-based operation while it is plainly ours to signal.
var localHosts = map[string]bool{
	"":          true,
	"127.0.0.1": true,
	"localhost": true,
	"::1":       true,
	"[::1]":     true,
	"0.0.0.0":   true,
}

// IsRemote reports whether the process runs on another machine, and so must be
// stopped over a transport rather than with a local signal.
func (p Proc) IsRemote() bool { return !localHosts[strings.ToLower(p.Host)] }

// Manager tracks a set of launched processes and tears them down verifiably.
type Manager struct {
	mu    sync.Mutex
	procs []Proc
	seen  map[string]bool
}

// New returns an empty Manager.
func New() *Manager { return &Manager{seen: map[string]bool{}} }

// procKey deduplicates by host and pid, since a local and a remote process can
// share a pid number.
func procKey(host string, pid int) string { return host + ":" + strconv.Itoa(pid) }

// Track registers a local PID (deduplicated). PIDs <= 1 are ignored (0 =
// attached / unknown; 1 = init, never ours).
func (m *Manager) Track(pid int, label string) {
	m.TrackProc(Proc{PID: pid, Label: label})
}

// TrackProc registers a full process record (deduplicated by host+pid),
// carrying its data directory and remote host when known.
func (m *Manager) TrackProc(p Proc) {
	if p.PID <= 1 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := procKey(p.Host, p.PID)
	if m.seen[key] {
		return
	}
	m.seen[key] = true
	m.procs = append(m.procs, p)
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

// StopAll terminates every tracked LOCAL process and verifies it is gone: it
// sends SIGTERM, waits up to grace for a clean exit, SIGKILLs any survivor, then
// polls briefly and returns the PIDs still alive (leaks — expected to be empty).
// Remote processes are handled by StopRemote, since their liveness cannot be
// probed with a local signal. Safe to call more than once.
func (m *Manager) StopAll(grace time.Duration) []int {
	procs := localProcs(m.Tracked())

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

// StopOne terminates a single tracked LOCAL process and verifies it is gone,
// with the same SIGTERM -> grace -> SIGKILL escalation StopAll uses. It exists
// for fault injection: a spec that stops one validator to test quorum must not
// take down the rest of the network, which is all StopAll can do.
//
// An untracked pid is an error (chainbench only stops what it started). A
// process that is already gone is not — stopping twice is idempotent, and a node
// that crashed on its own should not fail the step that meant to stop it.
func (m *Manager) StopOne(pid int, grace time.Duration) error {
	var target Proc
	found := false
	for _, p := range m.Tracked() {
		if p.PID == pid && !p.IsRemote() {
			target, found = p, true
			break
		}
	}
	if !found {
		return fmt.Errorf("process: pid %d is not a tracked local process", pid)
	}
	if !Alive(target.PID) {
		return nil
	}

	_ = syscall.Kill(target.PID, syscall.SIGTERM)
	procs := []Proc{target}
	if waitAllGone(procs, grace) {
		return nil
	}
	_ = syscall.Kill(target.PID, syscall.SIGKILL)
	if waitAllGone(procs, 5*time.Second) {
		return nil
	}
	return fmt.Errorf("process: pid %d (%s) still alive after SIGKILL", target.PID, target.Label)
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

// localProcs returns the subset of procs that run on the local host.
func localProcs(all []Proc) []Proc {
	out := make([]Proc, 0, len(all))
	for _, p := range all {
		if !p.IsRemote() {
			out = append(out, p)
		}
	}
	return out
}

// StopRemote stops every tracked remote process using kill (e.g. an SSH kill),
// returning any errors. Remote liveness cannot be probed with a local signal,
// so this is best-effort by design; the caller verifies via the transport.
func (m *Manager) StopRemote(kill func(host string, pid int) error) []error {
	var errs []error
	for _, p := range m.Tracked() {
		if !p.IsRemote() {
			continue
		}
		if err := kill(p.Host, p.PID); err != nil {
			errs = append(errs, fmt.Errorf("process: stop %s pid %d: %w", p.Host, p.PID, err))
		}
	}
	return errs
}

// DataDirs returns the unique, non-empty data directories of tracked processes,
// so they can be removed separately from stopping (design S2).
func (m *Manager) DataDirs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	seen := map[string]bool{}
	var out []string
	for _, p := range m.procs {
		if p.DataDir == "" || seen[p.DataDir] {
			continue
		}
		seen[p.DataDir] = true
		out = append(out, p.DataDir)
	}
	return out
}

// RemoveDataDirs deletes each tracked data directory. Stopping a node and
// removing its datadir are separate operations; call this only after the nodes
// are stopped (design S2). It returns any removal errors.
func (m *Manager) RemoveDataDirs() []error {
	var errs []error
	for _, dir := range m.DataDirs() {
		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, fmt.Errorf("process: remove datadir %s: %w", dir, err))
		}
	}
	return errs
}
