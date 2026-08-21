package keyring

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/accounts/keystore"
)

// Default scrypt parameters for keystore encryption (standard strength). A test
// or a fast path passes lighter values on KeystoreBackend.
const (
	defaultScryptN = 1 << 18
	defaultScryptP = 1
)

// Backend is how a key is persisted under a ring directory and read back.
//
// The two forms — a raw hex file and an encrypted keystore — differ only here,
// so a caller picks a backend and stays format-agnostic. It is the same choice
// cosmos-sdk spells --keyring-backend.
type Backend interface {
	// Save writes the key under name and returns the file path.
	Save(dir, name string, key PrivateKey, pw PasswordSource) (string, error)
	// Load reads a previously saved key by name.
	Load(dir, name string, pw PasswordSource) (PrivateKey, error)
}

// RawFileBackend stores the key as 0x-hex in <dir>/<name>.key (0600). No
// password is involved, which is why it is only appropriate for a throwaway
// local ring.
type RawFileBackend struct{}

// Save writes the hex key.
func (RawFileBackend) Save(dir, name string, key PrivateKey, _ PasswordSource) (string, error) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name+".key")
	if err := os.WriteFile(path, []byte("0x"+key.Hex()), secretPerm); err != nil {
		return "", err
	}
	return path, nil
}

// Load reads the hex key.
func (RawFileBackend) Load(dir, name string, _ PasswordSource) (PrivateKey, error) {
	return FileSource{Path: filepath.Join(dir, name+".key")}.Resolve(context.Background())
}

// KeystoreBackend stores an encrypted v3 keystore JSON in <dir>/<name>.json
// (0600), guarded by a password. ScryptN/ScryptP default to standard strength.
type KeystoreBackend struct {
	ScryptN int
	ScryptP int
}

// Save encrypts the key into a keystore JSON.
func (b KeystoreBackend) Save(dir, name string, key PrivateKey, pw PasswordSource) (string, error) {
	if pw == nil {
		return "", fmt.Errorf("keyring: keystore backend needs a password")
	}
	password, err := pw.Password()
	if err != nil {
		return "", err
	}
	n, p := b.ScryptN, b.ScryptP
	if n == 0 {
		n = defaultScryptN
	}
	if p == 0 {
		p = defaultScryptP
	}
	keyjson, err := keystore.Encrypt(key.Bytes(), password, n, p)
	if err != nil {
		return "", fmt.Errorf("keyring: encrypt keystore: %w", err)
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, keyjson, secretPerm); err != nil {
		return "", err
	}
	return path, nil
}

// Load decrypts a stored keystore.
func (KeystoreBackend) Load(dir, name string, pw PasswordSource) (PrivateKey, error) {
	return FileSource{Path: filepath.Join(dir, name+".json"), Password: pw}.Resolve(context.Background())
}
