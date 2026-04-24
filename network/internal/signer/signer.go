// Package signer is the chainbench-net signing boundary. Private key material
// lives ONLY inside sealed structs of this package; there is no accessor or
// reflection-visible export.
//
// Key material enters the process exclusively via env vars at spawn time:
//
//	CHAINBENCH_SIGNER_<ALIAS>_KEY=0x<64-hex-chars>
//
// Load() resolves an alias by uppercasing it and reading the matching env var.
// The returned Signer exposes Address() and SignTx() — no Export(), no
// GetKey(), no serialization path. slog.LogValuer on the sealed struct
// renders "***" to prevent accidental disclosure through structured logging.
package signer

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"os"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// Alias is the operator-chosen name used to look up key material in env.
type Alias string

// Signer is the minimal signing contract exposed to handlers. There is no
// accessor for the underlying private key bytes.
type Signer interface {
	Address() common.Address
	SignTx(ctx context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
}

// Sentinel errors for Load. Error messages from this package reference the
// alias and env var name only — never the raw key value.
var (
	ErrUnknownAlias = errors.New("signer: unknown alias")
	ErrInvalidAlias = errors.New("signer: alias must match [A-Za-z][A-Za-z0-9_]*")
	ErrInvalidKey   = errors.New("signer: key material is not a valid hex private key")
)

// aliasRE: POSIX-identifier-compatible so the resulting env var name (e.g.
// CHAINBENCH_SIGNER_ALICE_KEY) is accepted by common tooling (shells, docker
// -e, systemd EnvironmentFile, k8s ConfigMap). ASCII-only by RE2 construction;
// strings.ToUpper below is safe because the regex filters Unicode out before
// upper-casing.
var aliasRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// sealed holds the private key material. Field is unexported; no method
// returns the key bytes. LogValue / String / GoString redact for any
// fmt.* or slog.* consumer.
type sealed struct {
	alias Alias
	addr  common.Address
	key   *ecdsa.PrivateKey
}

func (s *sealed) Address() common.Address { return s.addr }

func (s *sealed) SignTx(_ context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
	if tx == nil {
		return nil, fmt.Errorf("signer.SignTx: tx is nil")
	}
	if chainID == nil {
		return nil, fmt.Errorf("signer.SignTx: chainID is nil")
	}
	signed, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), s.key)
	if err != nil {
		// err from go-ethereum is not expected to carry the key; defensive
		// wrap references the alias only.
		return nil, fmt.Errorf("signer.SignTx(%s): %w", s.alias, err)
	}
	return signed, nil
}

// LogValue implements slog.LogValuer so any structured-log attr containing a
// Signer renders as "***" instead of the underlying struct.
func (*sealed) LogValue() slog.Value { return slog.StringValue("***") }

// String / GoString implement the fmt.Stringer and fmt.GoStringer contracts
// so %v, %+v, %#v, %s all render as the redacted placeholder.
func (*sealed) String() string   { return "<signer:***>" }
func (*sealed) GoString() string { return "<signer:***>" }

// Load resolves alias → Signer via env var. Returns ErrInvalidAlias for
// structurally bad names, ErrUnknownAlias when no env var is set,
// ErrInvalidKey when the env value is not a valid hex private key.
func Load(alias Alias) (Signer, error) {
	a := string(alias)
	if a == "" || !aliasRE.MatchString(a) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidAlias, a)
	}
	envName := "CHAINBENCH_SIGNER_" + strings.ToUpper(a) + "_KEY"
	raw := os.Getenv(envName)
	if raw == "" {
		return nil, fmt.Errorf("%w: %s (env %s not set)", ErrUnknownAlias, a, envName)
	}
	hex := strings.TrimPrefix(raw, "0x")
	// Validate without ever embedding the raw value in an error message.
	key, err := crypto.HexToECDSA(hex)
	if err != nil {
		return nil, fmt.Errorf("%w: alias=%s (env %s)", ErrInvalidKey, a, envName)
	}
	addr := crypto.PubkeyToAddress(key.PublicKey)
	return &sealed{alias: alias, addr: addr, key: key}, nil
}
