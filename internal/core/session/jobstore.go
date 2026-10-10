package session

import (
	"os"
	"path/filepath"
)

// JobStore persists the control service's plans, jobs and audit in one snapshot.
// A service owns one store per directory; node files live elsewhere.
type JobStore struct{ root string }

func OpenJobStore(root string) (*JobStore, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	return &JobStore{root: root}, nil
}

func (s *JobStore) Read() ([]byte, error) { return os.ReadFile(filepath.Join(s.root, "jobs.json")) }

// Write publishes only a complete synced private snapshot.
func (s *JobStore) Write(b []byte) error {
	f, err := os.CreateTemp(s.root, ".jobs-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(s.root, "jobs.json"))
}
