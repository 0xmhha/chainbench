package collector

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// FileStore is a Store persisted to a JSON file, so run results survive across
// separate CLI invocations (requirement #17). It wraps a MemStore and rewrites
// the file after each SaveRun; reads are served from memory.
type FileStore struct {
	path string
	mu   sync.Mutex
	mem  *MemStore
}

// NewFileStore opens (or creates) a store at path. An existing file is loaded;
// a missing file yields an empty store.
func NewFileStore(path string) (*FileStore, error) {
	f := &FileStore{path: path, mem: NewMemStore()}
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return f, nil
	}
	if err != nil {
		return nil, fmt.Errorf("collector: read run store %s: %w", path, err)
	}
	var runs []RunRecord
	if len(b) > 0 {
		if err := json.Unmarshal(b, &runs); err != nil {
			return nil, fmt.Errorf("collector: parse run store %s: %w", path, err)
		}
	}
	for _, r := range runs {
		_ = f.mem.SaveRun(r)
	}
	return f, nil
}

// SaveRun upserts the record and rewrites the file.
func (f *FileStore) SaveRun(r RunRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	_ = f.mem.SaveRun(r)
	return f.flush()
}

// GetRun returns the record for id.
func (f *FileStore) GetRun(id string) (RunRecord, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mem.GetRun(id)
}

// ListRuns returns all records sorted by StartedAt, then ID.
func (f *FileStore) ListRuns() []RunRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.mem.ListRuns()
}

// flush writes the current records to disk (caller holds the lock).
func (f *FileStore) flush() error {
	if err := os.MkdirAll(filepath.Dir(f.path), 0o755); err != nil {
		return fmt.Errorf("collector: mkdir run store dir: %w", err)
	}
	b, err := json.MarshalIndent(f.mem.ListRuns(), "", "  ")
	if err != nil {
		return fmt.Errorf("collector: marshal run store: %w", err)
	}
	if err := os.WriteFile(f.path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("collector: write run store %s: %w", f.path, err)
	}
	return nil
}
