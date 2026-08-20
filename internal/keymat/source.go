package keymat

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/hdwallet"

	"github.com/0xmhha/chainbench/internal/core/provision"
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

// FileSource imports an account from a key file — a raw hex private key or an
// encrypted keystore JSON (decrypted with Password).
//
// The file may live on this machine or on a host reached over SSH: Files
// decides which, and nothing above this type has to know. A nil Files reads
// locally, so the common case stays a two-field literal.
//
// This replaces the former FileSource/RemoteFileSource pair. They differed only
// in how the bytes arrived, and keeping two types meant keeping two code paths
// that could drift — the remote one had grown its own SSH read.
type FileSource struct {
	// Files is where the key file lives. Nil means the local filesystem.
	Files provision.FileStore
	// Path is the key file's path on that store.
	Path string
	// Password decrypts a keystore; unused for a raw hex key.
	Password PasswordSource
}

// Resolve reads the key file and builds its account, detecting a keystore JSON
// versus a raw hex private key.
func (s FileSource) Resolve(ctx context.Context) (*account.Account, error) {
	files := s.Files
	if files == nil {
		files = provision.LocalFileStore{}
	}
	data, err := files.Read(ctx, s.Path)
	if err != nil {
		return nil, fmt.Errorf("keymat: read key file: %w", err)
	}
	return accountFromKeyBytes(data, s.Password)
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
