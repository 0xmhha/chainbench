package keyring

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

// DefaultCoinType is Ethereum's SLIP-44 coin type. BIP-39/BIP-44 uses it by
// convention for Ethereum-style chains, but a chain may register its own, so it
// is a knob rather than a hard-coded 60.
const DefaultCoinType uint32 = 60

// HDPath is a BIP-44 derivation path m/44'/coin'/account'/change/index. Only
// mnemonic sources use it. CoinType is first-class because the derived key
// depends on it: a chain that is not Ethereum should set its own so the
// addresses match that chain's wallets.
type HDPath struct {
	CoinType uint32
	Account  uint32
	Change   uint32
	Index    uint32
}

// DefaultHDPath returns m/44'/60'/0'/0/0 — the Ethereum default.
func DefaultHDPath() HDPath { return HDPath{CoinType: DefaultCoinType} }

// String renders the path in the standard notation the hdwallet consumes.
func (p HDPath) String() string {
	return fmt.Sprintf("m/44'/%d'/%d'/%d/%d", p.CoinType, p.Account, p.Change, p.Index)
}

// Source is where a key comes from: freshly generated, given literally, derived
// from a mnemonic, or read from a file here or on another host.
//
// Every source yields a [PrivateKey] and nothing else. What that key means —
// address, devp2p public key, BLS material — is [Derive]'s job, so a source
// never has to know which of those a caller wants.
//
// ctx is accepted for sources that do I/O; the rest ignore it.
type Source interface {
	Resolve(ctx context.Context) (PrivateKey, error)
}

// RandomSource generates a fresh key.
type RandomSource struct {
	// Rand supplies the entropy; nil uses crypto/rand.
	Rand func() ([]byte, error)
}

// Resolve generates a new key.
func (s RandomSource) Resolve(context.Context) (PrivateKey, error) {
	if s.Rand != nil {
		raw, err := s.Rand()
		if err != nil {
			return PrivateKey{}, err
		}
		return ParsePrivateKey(hex.EncodeToString(raw))
	}
	a, err := account.Generate()
	if err != nil {
		return PrivateKey{}, fmt.Errorf("keyring: generate: %w", err)
	}
	return fromAccount(a)
}

// PrivateKeySource takes a key the caller already holds, as 0x-hex or bare hex.
type PrivateKeySource struct {
	Hex string
}

// Resolve decodes the key.
func (s PrivateKeySource) Resolve(context.Context) (PrivateKey, error) {
	return ParsePrivateKey(s.Hex)
}

// MnemonicSource derives a key from a BIP-39 mnemonic at an HD path.
// Passphrase is the optional BIP-39 passphrase (the "25th word").
type MnemonicSource struct {
	Mnemonic   string
	Passphrase string
	Path       HDPath
}

// Resolve derives the key from the mnemonic at the configured path.
func (s MnemonicSource) Resolve(context.Context) (PrivateKey, error) {
	w, err := hdwallet.FromMnemonic(strings.TrimSpace(s.Mnemonic), s.Passphrase)
	if err != nil {
		return PrivateKey{}, fmt.Errorf("keyring: mnemonic: %w", err)
	}
	a, err := w.Derive(s.Path.String())
	if err != nil {
		return PrivateKey{}, fmt.Errorf("keyring: derive %s: %w", s.Path, err)
	}
	return fromAccount(a)
}

// FileSource reads a key file — a raw hex key, or an encrypted keystore JSON
// decrypted with Password.
//
// The file may be on this machine or on a host reached over SSH: Files decides
// which, and nothing above this type has to know. A nil Files reads locally, so
// the common case stays a two-field literal and local is not spelled as a
// special kind of remote.
type FileSource struct {
	// Files is where the key file lives. Nil means the local filesystem.
	Files provision.FileStore
	// Path is the key file's path on that store.
	Path string
	// Password decrypts a keystore; unused for a raw hex key.
	Password PasswordSource
}

// Resolve reads the key file and decodes it, detecting a keystore JSON versus a
// raw hex key.
func (s FileSource) Resolve(ctx context.Context) (PrivateKey, error) {
	files := s.Files
	if files == nil {
		files = provision.LocalFileStore{}
	}
	data, err := files.Read(ctx, s.Path)
	if err != nil {
		return PrivateKey{}, fmt.Errorf("keyring: read key file: %w", err)
	}
	return decodeKeyFile(data, s.Password)
}

// decodeKeyFile decodes key-file bytes, detecting a keystore JSON (decrypted
// with pw) versus a raw hex key.
func decodeKeyFile(data []byte, pw PasswordSource) (PrivateKey, error) {
	data = bytes.TrimSpace(data)
	if len(data) > 0 && data[0] == '{' {
		if pw == nil {
			return PrivateKey{}, fmt.Errorf("keyring: keystore needs a password")
		}
		password, err := pw.Password()
		if err != nil {
			return PrivateKey{}, err
		}
		a, err := account.FromKeystore(data, password)
		if err != nil {
			return PrivateKey{}, fmt.Errorf("keyring: decrypt keystore: %w", err)
		}
		return fromAccount(a)
	}
	return ParsePrivateKey(string(data))
}

// fromAccount extracts the key from an SDK account. The SDK's account type is
// the accounts library's, and it stops here: above this package a key is a
// [PrivateKey] and what it implies is an [Identity].
func fromAccount(a *account.Account) (PrivateKey, error) {
	return ParsePrivateKey(hex.EncodeToString(a.PrivateKeyBytes()))
}
