package keymat

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/hdwallet"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// Source resolves key material to an account, regardless of where it comes from.
// ctx is accepted for sources that do I/O (a remote file); pure sources ignore
// it. Adding a new origin means adding a Source — the callers do not change.
type Source interface {
	Resolve(ctx context.Context) (*account.Account, error)
}

// RandomSource generates a fresh random account.
type RandomSource struct{}

// Resolve generates a new keypair.
func (RandomSource) Resolve(context.Context) (*account.Account, error) {
	return account.Generate()
}

// PrivateKeySource imports a known private key given as 0x-hex (or bare hex).
type PrivateKeySource struct {
	Hex string
}

// Resolve decodes the private key and builds its account.
func (s PrivateKeySource) Resolve(context.Context) (*account.Account, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(s.Hex), "0x"))
	if err != nil {
		return nil, fmt.Errorf("keymat: invalid private key hex: %w", err)
	}
	return account.FromPrivateKeyBytes(b)
}

// MnemonicSource imports an account derived from a BIP-39 mnemonic at an HD path.
// Passphrase is the optional BIP-39 passphrase (the "25th word"); Path selects
// the derived account (its CoinType should match the target chain).
type MnemonicSource struct {
	Mnemonic   string
	Passphrase string
	Path       HDPath
}

// Resolve derives the account from the mnemonic at the configured path.
func (s MnemonicSource) Resolve(context.Context) (*account.Account, error) {
	w, err := hdwallet.FromMnemonic(strings.TrimSpace(s.Mnemonic), s.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("keymat: mnemonic: %w", err)
	}
	return w.Derive(s.Path.String())
}

// FileSource imports an account from a local file: a raw hex private key, or an
// encrypted keystore JSON (decrypted with Password).
type FileSource struct {
	Path     string
	Password PasswordSource
}

// Resolve reads the file and detects its form (keystore JSON vs raw hex).
func (s FileSource) Resolve(_ context.Context) (*account.Account, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		return nil, fmt.Errorf("keymat: read key file: %w", err)
	}
	return accountFromKeyBytes(data, s.Password)
}

// RemoteFileSource imports an account from a key file on a remote SSH host (raw
// hex or keystore). It reads the file over SSH, then applies the same detection
// as FileSource. Creds are the fully-resolved SSH credentials — the caller
// resolves them once, from the environment (ad-hoc host:path) or a server
// inventory (--server N). Read is injectable for testing; nil uses the real SSH
// read. Env supplies the host-key policy; nil uses os.Getenv.
type RemoteFileSource struct {
	Creds    remote.Credentials
	Path     string
	Password PasswordSource
	Read     func(ctx context.Context) ([]byte, error)
	Env      func(string) string
}

// Resolve reads the remote key file and builds its account.
func (s RemoteFileSource) Resolve(ctx context.Context) (*account.Account, error) {
	read := s.Read
	if read == nil {
		read = s.sshRead
	}
	data, err := read(ctx)
	if err != nil {
		return nil, err
	}
	return accountFromKeyBytes(data, s.Password)
}

// sshRead reads the remote file over SSH using the resolved credentials.
func (s RemoteFileSource) sshRead(ctx context.Context) ([]byte, error) {
	env := s.Env
	if env == nil {
		env = os.Getenv
	}
	hostKey, err := remote.ResolveHostKeyCallback(env)
	if err != nil {
		return nil, err
	}
	return remote.ReadFile(ctx, s.Creds, hostKey, s.Path)
}

// accountFromKeyBytes builds an account from raw key-file bytes, detecting a
// keystore JSON (decrypted with pw) versus a raw hex private key.
func accountFromKeyBytes(data []byte, pw PasswordSource) (*account.Account, error) {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '{' {
		if pw == nil {
			return nil, fmt.Errorf("keymat: keystore needs a password")
		}
		password, err := pw.Password()
		if err != nil {
			return nil, err
		}
		return account.FromKeystore(data, password)
	}
	return PrivateKeySource{Hex: string(data)}.Resolve(context.Background())
}
