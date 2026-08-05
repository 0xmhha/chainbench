package obs

import (
	"sort"
	"sync"
	"time"
)

// RunStatus is the terminal (or in-progress) state of a pipeline run.
type RunStatus string

const (
	RunRunning   RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
	// RunSkipped is a run that did not execute (e.g. a test case gated out by
	// chain compatibility or a missing capability). It is distinct from
	// RunSucceeded so a skipped case is never counted as a pass — the caller
	// (report/dashboard) can surface coverage rather than false green.
	RunSkipped RunStatus = "skipped"
)

// RunRecord is the stored state/result of one pipeline run (a setup, verify, or
// test invocation). It is what the dashboard's per-phase views and `report`
// read back.
type RunRecord struct {
	ID        string         `json:"id"`
	Phase     Phase          `json:"phase"`
	Chain     string         `json:"chain"`
	Network   string         `json:"network"`
	Status    RunStatus      `json:"status"`
	StartedAt time.Time      `json:"started_at"`
	EndedAt   time.Time      `json:"ended_at,omitempty"`
	Result    map[string]any `json:"result,omitempty"`
}

// Store persists run records. G0 ships an in-memory implementation; a
// file/embedded-KV backing lands with chainbenchd (G8) behind this same
// interface.
type Store interface {
	SaveRun(r RunRecord) error
	GetRun(id string) (RunRecord, bool)
	ListRuns() []RunRecord
}

// MemStore is an in-memory Store, safe for concurrent use. ListRuns returns
// records sorted by StartedAt (then ID) for stable output.
type MemStore struct {
	mu   sync.Mutex
	runs map[string]RunRecord
}

// NewMemStore returns an empty in-memory store.
func NewMemStore() *MemStore {
	return &MemStore{runs: map[string]RunRecord{}}
}

// SaveRun inserts or replaces the record for r.ID.
func (s *MemStore) SaveRun(r RunRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[r.ID] = r
	return nil
}

// GetRun returns the record for id and whether it exists.
func (s *MemStore) GetRun(id string) (RunRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r, ok := s.runs[id]
	return r, ok
}

// ListRuns returns all records sorted by StartedAt, then ID.
func (s *MemStore) ListRuns() []RunRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]RunRecord, 0, len(s.runs))
	for _, r := range s.runs {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].StartedAt.Equal(out[j].StartedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].StartedAt.Before(out[j].StartedAt)
	})
	return out
}
