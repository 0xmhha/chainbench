package session

import (
	"errors"
	"os"
	"path/filepath"
)

// AccountStore owns private Web account snapshots, setup capability and audit log.
// Authentication policy and password/session semantics stay in internal/app.
type AccountStore struct{ root string }

func OpenAccountStore(root string) (*AccountStore, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	return &AccountStore{root}, nil
}
func (s *AccountStore) ReadUsers() ([]byte, error) {
	path := filepath.Join(s.root, "users.json")
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("accounts require private permissions")
	}
	return os.ReadFile(path)
}
func (s *AccountStore) WriteUsers(b []byte) error {
	f, err := os.CreateTemp(s.root, ".users-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err = f.Write(b); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(s.root, "users.json"))
}
func (s *AccountStore) SetupToken(candidate string) (string, error) {
	path := filepath.Join(s.root, "setup.token")
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		f, e := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return "", e
		}
		_, err = f.WriteString(candidate)
		closeErr := f.Close()
		if err != nil {
			return "", err
		}
		if closeErr != nil {
			return "", closeErr
		}
		b = []byte(candidate)
	} else if err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm()&0077 != 0 || len(b) != 64 {
		return "", errors.New("invalid private setup token")
	}
	return string(b), nil
}
func (s *AccountStore) RemoveSetupToken() error {
	return os.Remove(filepath.Join(s.root, "setup.token"))
}
func (s *AccountStore) AppendAudit(b []byte) error {
	f, err := os.OpenFile(filepath.Join(s.root, "security-audit.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if _, err = f.Write(b); err != nil {
		return err
	}
	return f.Sync()
}
