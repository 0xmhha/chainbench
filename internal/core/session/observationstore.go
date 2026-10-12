package session

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

// ObservationStore archives collected node metrics, gaps and log lines per
// network. Records are only appended; nothing here expires or rewrites them.
// Only the collection cursor (offsets and last round) is replaced atomically.
type ObservationStore struct{ root string }

var observationName = regexp.MustCompile(`^[A-Za-z0-9_-][A-Za-z0-9_.-]*$`)

// ErrObservationName rejects a network or record name that could leave the
// archive directory.
var ErrObservationName = errors.New("invalid observation record name")

func OpenObservationStore(root string) (*ObservationStore, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	return &ObservationStore{root: root}, nil
}

func (s *ObservationStore) path(network, record string) (string, error) {
	dir, name := filepath.Split(record)
	if !observationName.MatchString(network) || (dir != "" && dir != "logs/") || !observationName.MatchString(name) {
		return "", ErrObservationName
	}
	return filepath.Join(s.root, network, dir, name), nil
}

// Append adds complete lines to a record such as "samples.jsonl" or
// "logs/node1.jsonl".
func (s *ObservationStore) Append(network, record string, b []byte) error {
	if len(b) == 0 {
		return nil
	}
	path, err := s.path(network, record)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err = f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// Open reads one record. A record that was never written reads as empty.
func (s *ObservationStore) Open(network, record string) (io.ReadCloser, error) {
	path, err := s.path(network, record)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return io.NopCloser(eofReader{}), nil
	}
	return f, err
}

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) { return 0, io.EOF }

// ReadCursor returns the collection cursor, or nil before the first round.
func (s *ObservationStore) ReadCursor(network string) ([]byte, error) {
	path, err := s.path(network, "state.json")
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return b, err
}

// WriteCursor publishes a complete synced cursor.
func (s *ObservationStore) WriteCursor(network string, b []byte) error {
	path, err := s.path(network, "state.json")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".state-*")
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
	return os.Rename(name, path)
}

// Replace atomically publishes a rewritten record. It exists for audited
// administrative deletion of one run's observations; collection only appends.
func (s *ObservationStore) Replace(network, record string, b []byte) error {
	path, err := s.path(network, record)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".replace-*")
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
	return os.Rename(name, path)
}

// Logs lists the archived log record names of a network, such as
// "logs/node1-20261009.jsonl", including nodes no longer in the chain record.
func (s *ObservationStore) Logs(network string) ([]string, error) {
	if !observationName.MatchString(network) {
		return nil, ErrObservationName
	}
	entries, err := os.ReadDir(filepath.Join(s.root, network, "logs"))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, entry := range entries {
		if !entry.IsDir() && observationName.MatchString(entry.Name()) && filepath.Ext(entry.Name()) == ".jsonl" {
			out = append(out, "logs/"+entry.Name())
		}
	}
	return out, nil
}
