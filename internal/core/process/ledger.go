package process

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// LedgerFile is the file a Ledger persists to inside its directory.
const LedgerFile = "process.json"

// Ledger is the persisted run ledger: which machine runs which binary, under
// which command, as which pid — keyed by label, surviving between commands so
// a later invocation (a stop, a health check, a pre-launch check) reaches the
// same processes an earlier one started.
//
// It records facts; it does not signal. Stopping is the Manager's or a
// driver's job, and a caller that stopped a process clears its entry.
type Ledger struct {
	mu    sync.Mutex
	dir   string
	procs map[string]Proc
}

// ledgerFile is the persisted shape.
type ledgerFile struct {
	Procs []Proc `json:"procs"`
}

// OpenLedger loads the ledger persisted in dir, or starts an empty one when
// none exists yet.
func OpenLedger(dir string) (*Ledger, error) {
	l := &Ledger{dir: dir, procs: map[string]Proc{}}
	raw, err := os.ReadFile(filepath.Join(dir, LedgerFile))
	if errors.Is(err, fs.ErrNotExist) {
		return l, nil
	}
	if err != nil {
		return nil, fmt.Errorf("process: read ledger: %w", err)
	}
	var f ledgerFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("process: parse ledger %s: %w", filepath.Join(dir, LedgerFile), err)
	}
	for _, p := range f.Procs {
		l.procs[p.Label] = p
	}
	return l, nil
}

// Save persists the ledger. Entries are sorted so the file is deterministic.
func (l *Ledger) Save() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	f := ledgerFile{Procs: make([]Proc, 0, len(l.procs))}
	for _, p := range l.procs {
		f.Procs = append(f.Procs, p)
	}
	sort.Slice(f.Procs, func(i, j int) bool { return f.Procs[i].Label < f.Procs[j].Label })
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("process: encode ledger: %w", err)
	}
	if err := os.MkdirAll(l.dir, 0o755); err != nil {
		return fmt.Errorf("process: ledger dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(l.dir, LedgerFile), b, 0o644); err != nil {
		return fmt.Errorf("process: write ledger: %w", err)
	}
	return nil
}

// Record enters a launched process under its label. A label whose previous
// entry is still recorded is a double launch — the caller asked to start
// something it already started — and is refused with both pids named.
func (l *Ledger) Record(p Proc) error {
	if p.Label == "" {
		return fmt.Errorf("process: record needs a label")
	}
	if p.PID <= 0 {
		return fmt.Errorf("process: record %s: pid %d is not a live process", p.Label, p.PID)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if prev, ok := l.procs[p.Label]; ok {
		return fmt.Errorf("process: %s already recorded as pid %d on %q — stop it (or clear the entry) before launching pid %d",
			p.Label, prev.PID, prev.Host, p.PID)
	}
	l.procs[p.Label] = p
	return nil
}

// Clear removes label's entry, returning what it was. Clearing an absent
// label is not an error: stop paths converge here after partial failures.
func (l *Ledger) Clear(label string) (Proc, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.procs[label]
	delete(l.procs, label)
	return p, ok
}

// Get returns label's recorded process.
func (l *Ledger) Get(label string) (Proc, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.procs[label]
	return p, ok
}

// Recorded returns every entry, sorted by label.
func (l *Ledger) Recorded() []Proc {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]Proc, 0, len(l.procs))
	for _, p := range l.procs {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}

// FindBinary returns the entries running the named binary on host, sorted by
// label — the pre-launch question "is one of these already running there?".
func (l *Ledger) FindBinary(host, binary string) []Proc {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []Proc
	for _, p := range l.procs {
		if p.Binary == binary && p.Host == host {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Label < out[j].Label })
	return out
}
