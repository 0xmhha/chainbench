package session

import (
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
)

// WebStore owns local control-plane deployment snapshots and their private encryption key.
// It never writes target node data. A process must open one owner per directory.
type WebStore struct{ root string }

// OpenWebStore prepares private local Web control-plane storage.
func OpenWebStore(root string) (*WebStore, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	return &WebStore{root: root}, nil
}

// CredentialKey returns a private AES key, refusing to replace a lost key for existing data.
func (s *WebStore) CredentialKey() ([]byte, error) {
	keyPath := filepath.Join(s.root, "credential.key")
	key, err := os.ReadFile(keyPath)
	if os.IsNotExist(err) {
		if _, err := os.Stat(filepath.Join(s.root, "deployment.json")); err == nil {
			return nil, errors.New("credential key missing for existing deployment data")
		}
		key = make([]byte, 32)
		if _, err = rand.Read(key); err != nil {
			return nil, err
		}
		f, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return nil, err
		}
		_, writeErr := f.Write(key)
		closeErr := f.Close()
		if writeErr != nil {
			return nil, writeErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
	} else if err != nil {
		return nil, err
	}
	info, err := os.Stat(keyPath)
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("credential key requires private permissions")
	}
	return key, nil
}

// ReadDeployment reads the Web control-plane snapshot.
func (s *WebStore) ReadDeployment() ([]byte, error) {
	return os.ReadFile(filepath.Join(s.root, "deployment.json"))
}

// WriteDeployment atomically publishes a complete private snapshot after syncing it.
func (s *WebStore) WriteDeployment(b []byte) error {
	f, err := os.CreateTemp(s.root, ".deployment-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
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
	return os.Rename(name, filepath.Join(s.root, "deployment.json"))
}
