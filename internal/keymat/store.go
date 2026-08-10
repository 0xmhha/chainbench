package keymat

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/accounts/account"
)

// Default scrypt parameters for keystore encryption (standard-strength). Tests
// or fast paths can pass lighter values via KeystoreStore.
const (
	defaultScryptN = 1 << 18
	defaultScryptP = 1
)

// keyFilePerm and dirPerm keep saved key material owner-only.
const (
	keyFilePerm os.FileMode = 0o600
	dirPerm     os.FileMode = 0o755
)

// Store persists an account's key material under a directory and reads it back.
// The two forms — a raw hex file and an encrypted keystore — differ only here,
// so callers pick a Store and stay format-agnostic.
type Store interface {
	// Save writes the account's material and returns the file path.
	Save(dir, name string, a *account.Account, pw PasswordSource) (string, error)
	// Load reads a previously saved account by name.
	Load(dir, name string, pw PasswordSource) (*account.Account, error)
}

// RawFileStore stores the private key as 0x-hex in <dir>/<name>.key (0600). No
// password is used.
type RawFileStore struct{}

// Save writes the hex private key.
func (RawFileStore) Save(dir, name string, a *account.Account, _ PasswordSource) (string, error) {
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name+".key")
	content := []byte("0x" + hex.EncodeToString(a.PrivateKeyBytes()))
	if err := os.WriteFile(path, content, keyFilePerm); err != nil {
		return "", err
	}
	return path, nil
}

// Load reads the hex private key.
func (RawFileStore) Load(dir, name string, _ PasswordSource) (*account.Account, error) {
	path := filepath.Join(dir, name+".key")
	return FileSource{Path: path}.Resolve(context.Background())
}

// KeystoreStore stores an encrypted keystore JSON in <dir>/<name>.json (0600),
// guarded by a password. ScryptN/ScryptP default to standard strength.
type KeystoreStore struct {
	ScryptN int
	ScryptP int
}

// Save encrypts the account into a keystore file.
func (s KeystoreStore) Save(dir, name string, a *account.Account, pw PasswordSource) (string, error) {
	if pw == nil {
		return "", fmt.Errorf("keymat: keystore store needs a password")
	}
	password, err := pw.Password()
	if err != nil {
		return "", err
	}
	n, p := s.ScryptN, s.ScryptP
	if n == 0 {
		n = defaultScryptN
	}
	if p == 0 {
		p = defaultScryptP
	}
	keyjson, err := a.ToKeystore(password, n, p)
	if err != nil {
		return "", fmt.Errorf("keymat: encrypt keystore: %w", err)
	}
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return "", err
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, keyjson, keyFilePerm); err != nil {
		return "", err
	}
	return path, nil
}

// Load decrypts the keystore file.
func (KeystoreStore) Load(dir, name string, pw PasswordSource) (*account.Account, error) {
	path := filepath.Join(dir, name+".json")
	return FileSource{Path: path, Password: pw}.Resolve(context.Background())
}
