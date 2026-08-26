package store

import (
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
)

// The model lives in the keyring package; storage reads and writes it. These
// aliases let this package's code speak the model's names unqualified — they
// introduce no new types.
type (
	Entry          = keyring.Entry
	Preset         = keyring.Preset
	Network        = keyring.Network
	Label          = keyring.Label
	PrivateKey     = derive.PrivateKey
	Identity       = derive.Identity
	Derivation     = derive.Derivation
	Source         = keyring.Source
	PasswordSource = keyring.PasswordSource
	BLS            = derive.BLS
	FileSource     = keyring.FileSource
)

const (
	AccountOnly = derive.AccountOnly
	WithBLS     = derive.WithBLS

	dirPerm    = keyring.DirPerm
	secretPerm = keyring.SecretPerm
	publicPerm = keyring.PublicPerm
)
