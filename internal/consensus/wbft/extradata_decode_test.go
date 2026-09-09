package wbft

import (
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
)

// addr20/bls48 build valid-length hex inputs for the encoder.
func addr20(b byte) string { return "0x" + strings.Repeat(hex.EncodeToString([]byte{b}), addressLen) }
func bls48(b byte) string  { return "0x" + strings.Repeat(hex.EncodeToString([]byte{b}), blsPubKeyLen) }

// TestGenesisValidators_RoundTripsTheEncoder is the core guarantee: the
// addresses ExtraData writes are the addresses GenesisValidators reads back, in
// order.
func TestGenesisValidators_RoundTripsTheEncoder(t *testing.T) {
	vals := []string{addr20(0x11), addr20(0x22), addr20(0x33), addr20(0x44)}
	bls := []string{bls48(0xa1), bls48(0xb2), bls48(0xc3), bls48(0xd4)}

	extra, err := ExtraData(vals, bls)
	if err != nil {
		t.Fatalf("ExtraData: %v", err)
	}
	genesis, err := json.Marshal(map[string]any{"extraData": extra, "config": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}

	got, err := genesisValidatorsFromExtraData(genesis)
	if err != nil {
		t.Fatalf("GenesisValidators: %v", err)
	}
	if len(got) != len(vals) {
		t.Fatalf("got %d validators, want %d", len(got), len(vals))
	}
	for i := range vals {
		if !strings.EqualFold(got[i], vals[i]) {
			t.Fatalf("validator %d = %s, want %s", i, got[i], vals[i])
		}
	}
}

// TestGenesisValidators_ManyValidatorsCrossTheLongFormBoundary: with enough
// validators the candidate list and the whole extra-data exceed 55 bytes, so the
// decoder's long-form length headers are exercised.
func TestGenesisValidators_ManyValidatorsCrossTheLongFormBoundary(t *testing.T) {
	const n = 15
	var vals, bls []string
	for i := 0; i < n; i++ {
		vals = append(vals, addr20(byte(i+1)))
		bls = append(bls, bls48(byte(i+1)))
	}
	extra, err := ExtraData(vals, bls)
	if err != nil {
		t.Fatal(err)
	}
	genesis := []byte(`{"extraData":"` + extra + `"}`)
	got, err := genesisValidatorsFromExtraData(genesis)
	if err != nil {
		t.Fatalf("GenesisValidators: %v", err)
	}
	if len(got) != n {
		t.Fatalf("got %d validators, want %d", len(got), n)
	}
	if !strings.EqualFold(got[0], vals[0]) || !strings.EqualFold(got[n-1], vals[n-1]) {
		t.Fatalf("boundary validators wrong: first=%s last=%s", got[0], got[n-1])
	}
}

func TestGenesisValidators_Errors(t *testing.T) {
	if _, err := genesisValidatorsFromExtraData([]byte(`{"config":{}}`)); err == nil {
		t.Error("a genesis with no extraData must error")
	}
	if _, err := genesisValidatorsFromExtraData([]byte(`not json`)); err == nil {
		t.Error("invalid JSON must error")
	}
	if _, err := genesisValidatorsFromExtraData([]byte(`{"extraData":"0xzz"}`)); err == nil {
		t.Error("non-hex extraData must error")
	}
	if _, err := genesisValidatorsFromExtraData([]byte(`{"extraData":"0xc0"}`)); err == nil {
		t.Error("an empty list is not the WBFTExtra shape and must error")
	}
}
