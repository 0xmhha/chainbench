package session

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

// HistoryStore owns captured evidence and deletion tombstones in one private
// snapshot. Removing a capture never touches an engine session or live data.
type HistoryStore struct{ root string }

func OpenHistoryStore(root string) (*HistoryStore, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	return &HistoryStore{root: root}, nil
}

func (s *HistoryStore) Read() ([]byte, error) {
	f, err := os.Open(filepath.Join(s.root, "history.json"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, (64<<20)+1))
	if len(b) > 64<<20 {
		return nil, errors.New("history snapshot exceeds storage limit")
	}
	return b, err
}

func (s *HistoryStore) Write(b []byte) error {
	if len(b) > 64<<20 {
		return errors.New("history snapshot exceeds storage limit")
	}
	f, err := os.CreateTemp(s.root, ".history-*")
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
	return os.Rename(name, filepath.Join(s.root, "history.json"))
}
