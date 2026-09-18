package wbft

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// WBFT genesis extra-data.
//
// The wbft family reads its validator set out of the genesis extra-data: an
// RLP-encoded WBFTExtra (go-stablenet core/types/istanbul.go). It is therefore
// consensus-critical and it belongs to this family — it used to live in the
// key generator, which meant a package about key material decided what a
// genesis says about block production.
//
// Genesis shape (verified byte-for-byte against keys/preset and against the
// chain's own simulated backend genExtraData):
//
//	WBFTExtra{
//	  VanityData:  nil, RandaoReveal: nil,
//	  PrevRound: 0, PrevPreparedSeal: nil, PrevCommittedSeal: nil,
//	  Round: 0, PreparedSeal: nil, CommittedSeal: nil,
//	  GasTip: params.InitialGasTip,
//	  EpochInfo: {
//	    Candidates:    [{addr, DefaultDiligence} ...],
//	    Validators:    [0 .. n-1],          // indices into Candidates
//	    BLSPublicKeys: [48-byte key ...],
//	  },
//	}
const (
	// initialGasTip mirrors go-stablenet params.InitialGasTip — the genesis
	// gas tip the chain expects in the extra-data (wei).
	initialGasTip = 27_600_000_000_000
	// defaultDiligence mirrors go-stablenet types.DefaultDiligence: 95% of the
	// 2*10^6 diligence ceiling, the starting weight of every genesis validator.
	defaultDiligence = 1_900_000
	// blsPubKeyLen is the BLS public key length the epoch info carries.
	blsPubKeyLen = 48
	// addressLen is an execution-layer address length.
	addressLen = 20
)

// ExtraData computes the genesis extra-data for a wbft-family chain from the
// validator addresses and their BLS public keys (0x-hex, index-aligned). The
// result is 0x-hex RLP.
//
// It is derived, never declared: extra-data that disagrees with the validator
// set passes genesis validation and then fails in consensus, where the cause is
// far from the symptom. [BuildGenesis] computes it rather than accepting one.
func ExtraData(validators, blsKeys []string) (string, error) {
	if len(validators) == 0 {
		return "", fmt.Errorf("wbft: extradata: no validators")
	}
	if len(validators) != len(blsKeys) {
		return "", fmt.Errorf("wbft: extradata: %d validators but %d BLS keys",
			len(validators), len(blsKeys))
	}
	candidates := make([]any, len(validators))
	indices := make([]any, len(validators))
	keys := make([]any, len(validators))
	for i, v := range validators {
		addr, err := hexBytes(v, addressLen)
		if err != nil {
			return "", fmt.Errorf("wbft: extradata: validator %d: %w", i, err)
		}
		bls, err := hexBytes(blsKeys[i], blsPubKeyLen)
		if err != nil {
			return "", fmt.Errorf("wbft: extradata: BLS key %d: %w", i, err)
		}
		candidates[i] = []any{addr, uint64(defaultDiligence)}
		indices[i] = uint64(i)
		keys[i] = bls
	}
	extra := []any{
		[]byte{},  // VanityData
		[]byte{},  // RandaoReveal
		uint64(0), // PrevRound
		[]any{},   // PrevPreparedSeal (nil)
		[]any{},   // PrevCommittedSeal (nil)
		uint64(0), // Round
		[]any{},   // PreparedSeal (nil)
		[]any{},   // CommittedSeal (nil)
		uint64(initialGasTip),
		[]any{candidates, indices, keys}, // EpochInfo
	}
	return "0x" + hex.EncodeToString(rlpEncode(extra)), nil
}

// hexBytes decodes a 0x-hex string and checks its byte length.
func hexBytes(s string, wantLen int) ([]byte, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, err
	}
	if len(b) != wantLen {
		return nil, fmt.Errorf("%d bytes, want %d", len(b), wantLen)
	}
	return b, nil
}

// rlpEncode is the minimal RLP encoder this file needs: byte strings, uint64,
// and nested lists. Kept here rather than importing a chain SDK — RLP's two
// forms are simpler than the dependency.
func rlpEncode(v any) []byte {
	switch x := v.(type) {
	case []byte:
		return rlpString(x)
	case uint64:
		return rlpString(rlpUint(x))
	case []any:
		var payload []byte
		for _, item := range x {
			payload = append(payload, rlpEncode(item)...)
		}
		return append(rlpLen(len(payload), 0xc0), payload...)
	default:
		panic(fmt.Sprintf("rlpEncode: unsupported type %T", v))
	}
}

// rlpString encodes a byte string with its header.
func rlpString(b []byte) []byte {
	if len(b) == 1 && b[0] < 0x80 {
		return b
	}
	return append(rlpLen(len(b), 0x80), b...)
}

// rlpUint is RLP's canonical integer form: big-endian, no leading zeros,
// zero encodes as the empty string.
func rlpUint(u uint64) []byte {
	if u == 0 {
		return nil
	}
	var b [8]byte
	n := 0
	for i := 7; i >= 0; i-- {
		b[i] = byte(u)
		u >>= 8
		n = 8 - i
		if u == 0 {
			break
		}
	}
	return b[8-n:]
}

// rlpLen encodes a length header with the given base (0x80 strings, 0xc0
// lists).
func rlpLen(n int, base byte) []byte {
	if n < 56 {
		return []byte{base + byte(n)}
	}
	size := rlpUint(uint64(n))
	return append([]byte{base + 55 + byte(len(size))}, size...)
}
