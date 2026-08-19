package keyring

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

// NodekeyLen is the byte length of a nodekey — a secp256k1 private key.
const NodekeyLen = 32

// ErrInvalidNodekey is returned for input that is not a usable nodekey: the
// wrong length, not hex, or zero. Callers match it with errors.Is.
var ErrInvalidNodekey = errors.New("invalid nodekey")

// redacted is what a Nodekey prints as. It names the type so a log line still
// says what was there, without saying what it was.
const redacted = "keyring.Nodekey(redacted)"

// Nodekey is the 32-byte secp256k1 secret a node is built from. Every public
// part of an identity derives from it, so it is the only value that has to stay
// secret and the only one that has to be stored.
//
// The bytes are unexported and the String and GoString methods redact them, so
// a Nodekey cannot reach a log or an error message by accident. Use [Nodekey.Hex]
// to disclose it deliberately.
type Nodekey struct {
	b [NodekeyLen]byte
}

// ParseNodekey decodes a hex nodekey, with or without a 0x prefix and ignoring
// surrounding whitespace. It rejects anything that is not exactly
// [NodekeyLen] bytes, and rejects an all-zero key: zero is not a valid
// secp256k1 scalar, and a node started with one has an identity nobody can
// sign for.
func ParseNodekey(s string) (Nodekey, error) {
	trimmed := strings.TrimSpace(s)
	trimmed = strings.TrimPrefix(trimmed, "0x")
	trimmed = strings.TrimPrefix(trimmed, "0X")

	// The error deliberately reports the length, never the input: an invalid
	// nodekey is still key material, and error strings travel into logs.
	if len(trimmed) != hex.EncodedLen(NodekeyLen) {
		return Nodekey{}, fmt.Errorf("%w: got %d hex digits, want %d",
			ErrInvalidNodekey, len(trimmed), hex.EncodedLen(NodekeyLen))
	}
	raw, err := hex.DecodeString(trimmed)
	if err != nil {
		return Nodekey{}, fmt.Errorf("%w: not hexadecimal", ErrInvalidNodekey)
	}
	var k Nodekey
	copy(k.b[:], raw)
	if k.isZero() {
		return Nodekey{}, fmt.Errorf("%w: all zero", ErrInvalidNodekey)
	}
	return k, nil
}

// NewNodekey reads a fresh nodekey from rand. Randomness is a parameter rather
// than a package-level default so that generation is reproducible under test;
// production callers pass crypto/rand.Reader.
//
// It retries on the (vanishingly unlikely) all-zero draw rather than returning
// an unusable key.
func NewNodekey(rand io.Reader) (Nodekey, error) {
	// Three attempts is already far beyond the probability of a zero draw; the
	// bound exists so a broken reader that returns zeros cannot spin forever.
	const attempts = 3
	var k Nodekey
	for range attempts {
		if _, err := io.ReadFull(rand, k.b[:]); err != nil {
			return Nodekey{}, fmt.Errorf("keyring: read entropy: %w", err)
		}
		if !k.isZero() {
			return k, nil
		}
	}
	return Nodekey{}, fmt.Errorf("keyring: entropy source produced only zero keys")
}

// Hex returns the nodekey as 64 lowercase hex digits with no 0x prefix — the
// on-disk form the chains read from a datadir's nodekey file.
//
// This is a deliberate disclosure of secret material. Every call site is a
// place the secret leaves the type, so they are worth grepping for.
func (k Nodekey) Hex() string { return hex.EncodeToString(k.b[:]) }

// Bytes returns a copy of the raw key. The copy matters: callers must not be
// able to mutate a Nodekey through the slice they were handed.
func (k Nodekey) Bytes() []byte {
	out := make([]byte, NodekeyLen)
	copy(out, k.b[:])
	return out
}

// String redacts the key. It exists so that fmt verbs cannot leak it.
func (Nodekey) String() string { return redacted }

// GoString redacts the key under %#v, which would otherwise print the byte
// array in full.
func (Nodekey) GoString() string { return redacted }

func (k Nodekey) isZero() bool {
	var zero [NodekeyLen]byte
	return k.b == zero
}
