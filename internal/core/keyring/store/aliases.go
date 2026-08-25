package store

import "github.com/0xmhha/chainbench/internal/core/keyring"

// The model lives in the keyring package; storage reads and writes it. These
// aliases let this package's code speak the model's names unqualified — they
// introduce no new types.
type (
	Entry          = keyring.Entry
	Preset         = keyring.Preset
	Network        = keyring.Network
	Label          = keyring.Label
	PrivateKey     = keyring.PrivateKey
	Identity       = keyring.Identity
	Derivation     = keyring.Derivation
	Source         = keyring.Source
	PasswordSource = keyring.PasswordSource
	BLS            = keyring.BLS
	FileSource     = keyring.FileSource
)

const (
	AccountOnly = keyring.AccountOnly
	WithBLS     = keyring.WithBLS

	dirPerm    = keyring.DirPerm
	secretPerm = keyring.SecretPerm
	publicPerm = keyring.PublicPerm
)
