package keyreg

import (
	"context"

	"github.com/0xmhha/chainbench/internal/core/driver"
)

// Key is one named key: a signing/account key or a node identity key. BLS and
// PoP are populated only for validator keys that need them.
type Key struct {
	Name    string
	Address string
	Private []byte
	BLS     []byte
	PoP     []byte
}

// Source selects how Ensure obtains a key.
type Source int

const (
	// Random generates a fresh random keypair.
	Random Source = iota
	// LocalFile copies an existing key from the local path given as ref.
	LocalFile
	// RemoteDownload reads an existing key from a remote host over SSH (ref).
	RemoteDownload
	// Literal registers key material the caller already holds, with ref as the
	// hex private key. It is how an existing key set the caller has decoded —
	// a preset's node identities — enters the registry without a second read.
	Literal
)

// BLSDeriver derives a BLS public key and proof-of-possession from a private
// key. Implementations delegate to the external bootnode binary, so ctx bounds
// that process execution and a missing binary is a clear error.
type BLSDeriver interface {
	Derive(ctx context.Context, private []byte) (bls, pop []byte, err error)
}

// EnsureOpts configures a single Ensure call. When NeedBLS is set, BLS must be
// non-nil; a missing deriver is an error rather than a key silently lacking BLS.
type EnsureOpts struct {
	NeedBLS bool
	BLS     BLSDeriver
	// ExpectAddress, when set, is the address the caller believes this key
	// belongs to (0x-hex, case-insensitive). Ensure derives the address from the
	// private key and fails on a mismatch, so a key set whose declared identity
	// has drifted from its key material is caught before a node launches with
	// one identity while the genesis registers another.
	ExpectAddress string
}

// Registry stores named keys for a session and materializes them on demand.
type Registry interface {
	// Ensure returns the named key, creating/copying/downloading it per src if
	// absent (idempotent). ctx bounds RemoteDownload's SSH I/O.
	Ensure(ctx context.Context, name string, src Source, ref string, opts EnsureOpts) (Key, error)
	// Get returns an already-registered key from memory.
	Get(name string) (Key, bool)
	// UploadTo ships the named keys to a remote host via fp.
	UploadTo(ctx context.Context, fp driver.FileProvisioner, names []string, remotePath string) error
}
