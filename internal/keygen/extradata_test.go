package keygen

import (
	"strings"
	"testing"
)

// The shipped preset's values, verbatim (keys/preset/metadata.json). The golden
// test reproduces its extra-data byte-for-byte from its own validator set,
// which pins both the RLP encoder and the WBFTExtra field layout at once.
var presetValidators = []string{
	"0xc17d493883eaa3b4cceb0f214b273392d562f9d8",
	"0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c",
	"0x8c4a10b9108d49b9d23f764464090831d9c17764",
	"0x8eb79036bc0f3aba136ef18b3a2fb8c1188939a6",
}

var presetBLSKeys = []string{
	"0xa00eb14731965f294993a2df1cf09e5b826193a41853fd9aaa7195922b8461c97b215a1181d4ddecc9f5981fdd47556f",
	"0x929af9896092b61db0ead8931feaed3f77058825c3c82f20fd9557a244b8732303f2136b6acd06ba7e1b861bf5514449",
	"0x8c7faed16ab71ca6a3f8d82d643f6502e4c2dc3ecf48e86ed4d5dba42e67240313b84e911ad1bbf5783263284f09c1d0",
	"0xa63e51dd59a291b3cef9804c0790a0f95285297fb4fe141a587c6dda0784822c27e6e6b9404754e679ae4cca62d0ce4a",
}

const presetExtraData = "0xf90147808080c0c080c0c086191a20322000f90135f868d994c17d493883eaa3b4cceb0f214b273392d562f9d8831cfde0d9942493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c831cfde0d9948c4a10b9108d49b9d23f764464090831d9c17764831cfde0d9948eb79036bc0f3aba136ef18b3a2fb8c1188939a6831cfde0c480010203f8c4b0a00eb14731965f294993a2df1cf09e5b826193a41853fd9aaa7195922b8461c97b215a1181d4ddecc9f5981fdd47556fb0929af9896092b61db0ead8931feaed3f77058825c3c82f20fd9557a244b8732303f2136b6acd06ba7e1b861bf5514449b08c7faed16ab71ca6a3f8d82d643f6502e4c2dc3ecf48e86ed4d5dba42e67240313b84e911ad1bbf5783263284f09c1d0b0a63e51dd59a291b3cef9804c0790a0f95285297fb4fe141a587c6dda0784822c27e6e6b9404754e679ae4cca62d0ce4a"

func TestWBFTExtraData_GoldenAgainstShippedPreset(t *testing.T) {
	got, err := WBFTExtraData(presetValidators, presetBLSKeys)
	if err != nil {
		t.Fatal(err)
	}
	if got != presetExtraData {
		t.Fatalf("extra-data does not reproduce the shipped preset:\n got %s\nwant %s", got, presetExtraData)
	}
}

func TestWBFTExtraData_SingleValidator(t *testing.T) {
	got, err := WBFTExtraData(presetValidators[:1], presetBLSKeys[:1])
	if err != nil {
		t.Fatal(err)
	}
	// Spot-check the structural landmarks rather than a full literal: the
	// gas tip constant, one candidate entry with the diligence weight, and
	// the single validator index (RLP uint 0 = empty string, 0x80).
	for _, frag := range []string{
		"191a20322000", // InitialGasTip
		"c17d493883eaa3b4cceb0f214b273392d562f9d8831cfde0", // addr + DefaultDiligence
		"c180", // Validators: [0]
	} {
		if !strings.Contains(got, frag) {
			t.Fatalf("extra-data missing %s:\n%s", frag, got)
		}
	}
}

func TestWBFTExtraData_Rejects(t *testing.T) {
	if _, err := WBFTExtraData(nil, nil); err == nil {
		t.Fatal("no validators must fail")
	}
	if _, err := WBFTExtraData(presetValidators, presetBLSKeys[:2]); err == nil {
		t.Fatal("count mismatch must fail")
	}
	if _, err := WBFTExtraData([]string{"0x1234"}, presetBLSKeys[:1]); err == nil {
		t.Fatal("short address must fail")
	}
	if _, err := WBFTExtraData(presetValidators[:1], []string{"0xdead"}); err == nil {
		t.Fatal("short BLS key must fail")
	}
}

func TestRLPEncodePrimitives(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{[]byte{}, "80"},                // empty string
		{uint64(0), "80"},               // zero = empty string
		{uint64(1), "01"},               // single byte < 0x80
		{uint64(1_900_000), "831cfde0"}, // diligence
		{[]any{}, "c0"},                 // empty list
		{[]any{uint64(1), uint64(2)}, "c20102"},
	}
	for _, tc := range cases {
		if got := hexOf(rlpEncode(tc.in)); got != tc.want {
			t.Errorf("rlpEncode(%v) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func hexOf(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, digits[c>>4], digits[c&0xf])
	}
	return string(out)
}
