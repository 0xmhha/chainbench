package derive

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// PrivateKeyLen is the byte length of a secp256k1 private key.
const PrivateKeyLen = 32

// ErrInvalidPrivateKey is returned for input that is not a usable key: the
// wrong length, not hex, or zero. Callers match it with errors.Is.
var ErrInvalidPrivateKey = errors.New("invalid private key")

// redacted is what a PrivateKey prints as. It names the type so a log line
// still says what was there, without saying what it was.
const redacted = "keyring.PrivateKey(redacted)"

// PrivateKey is a 32-byte secp256k1 secret. Everything public about the holder
// derives from it, so it is the only value that has to stay secret and the only
// one that has to be stored.
//
// The type is named for what it is rather than for a role it plays. The same
// kind of key is a node's identity in a datadir's nodekey file and an account's
// key in a keystore; on the wbft family they are even the same key. Naming it
// "nodekey" would have made every account read as a node.
//
// The bytes are unexported and the String and GoString methods redact them, so
// a key cannot reach a log or an error message by accident. Use
// [PrivateKey.Hex] to disclose it deliberately.
type PrivateKey struct {
	b [PrivateKeyLen]byte
}

// ParsePrivateKey decodes a hex key, with or without a 0x prefix and ignoring
// surrounding whitespace. It rejects anything that is not exactly
// [PrivateKeyLen] bytes, and rejects an all-zero key: zero is not a valid
// secp256k1 scalar, and a node started with one has an identity nobody can
// sign for.
func ParsePrivateKey(s string) (PrivateKey, error) {
	trimmed := strings.TrimSpace(s)
	trimmed = strings.TrimPrefix(trimmed, "0x")
	trimmed = strings.TrimPrefix(trimmed, "0X")

	// The error deliberately reports the length, never the input: an invalid
	// key is still key material, and error strings travel into logs.
	if len(trimmed) != hex.EncodedLen(PrivateKeyLen) {
		return PrivateKey{}, fmt.Errorf("%w: got %d hex digits, want %d",
			ErrInvalidPrivateKey, len(trimmed), hex.EncodedLen(PrivateKeyLen))
	}
	raw, err := hex.DecodeString(trimmed)
	if err != nil {
		return PrivateKey{}, fmt.Errorf("%w: not hexadecimal", ErrInvalidPrivateKey)
	}
	var k PrivateKey
	copy(k.b[:], raw)
	if k.isZero() {
		return PrivateKey{}, fmt.Errorf("%w: all zero", ErrInvalidPrivateKey)
	}
	return k, nil
}

// NewPrivateKey reads a fresh key from rand. Randomness is a parameter rather
// than a package-level default so that generation is reproducible under test;
// production callers pass crypto/rand.Reader.
//
// It retries on the (vanishingly unlikely) all-zero draw rather than returning
// an unusable key.
func NewPrivateKey(rand io.Reader) (PrivateKey, error) {
	// Three attempts is already far beyond the probability of a zero draw; the
	// bound exists so a broken reader that returns zeros cannot spin forever.
	const attempts = 3
	var k PrivateKey
	for range attempts {
		if _, err := io.ReadFull(rand, k.b[:]); err != nil {
			return PrivateKey{}, fmt.Errorf("keyring: read entropy: %w", err)
		}
		if !k.isZero() {
			return k, nil
		}
	}
	return PrivateKey{}, fmt.Errorf("keyring: entropy source produced only zero keys")
}

// Hex returns the key as 64 lowercase hex digits with no 0x prefix — the
// on-disk form the chains read from a datadir's nodekey file.
//
// This is a deliberate disclosure of secret material. Every call site is a
// place the secret leaves the type, so they are worth grepping for.
func (k PrivateKey) Hex() string { return hex.EncodeToString(k.b[:]) }

// Bytes returns a copy of the raw key. The copy matters: callers must not be
// able to mutate a PrivateKey through the slice they were handed.
func (k PrivateKey) Bytes() []byte {
	out := make([]byte, PrivateKeyLen)
	copy(out, k.b[:])
	return out
}

// String redacts the key. It exists so that fmt verbs cannot leak it.
func (PrivateKey) String() string { return redacted }

// GoString redacts the key under %#v, which would otherwise print the byte
// array in full.
func (PrivateKey) GoString() string { return redacted }

func (k PrivateKey) isZero() bool {
	var zero [PrivateKeyLen]byte
	return k.b == zero
}
