package accounts_test

import (
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

// z64 is 64 hex zeros (a zero 32-byte word).
const z64 = "0000000000000000000000000000000000000000000000000000000000000000"

func TestEncodeCallArgs_Static(t *testing.T) {
	// transfer(address,uint256): selector + address word + uint word.
	got := accounts.EncodeCallArgs("transfer(address,uint256)",
		accounts.Address("0x00000000000000000000000000000000000000ab"), accounts.Uint(big.NewInt(1)))
	want := "0xa9059cbb" +
		z64[:24] + "00000000000000000000000000000000000000ab" +
		z64[:63] + "1"
	if got != want {
		t.Errorf("static encode:\n got=%s\nwant=%s", got, want)
	}
}

func TestEncodeABI_DynamicString(t *testing.T) {
	// f(string) with "hi": head offset 0x20, then length 2, then "hi" padded.
	got := accounts.EncodeCallArgs("f(string)", accounts.StringArg("hi"))
	want := "0x" + sel4("f(string)") +
		z64[:62] + "20" + // offset = 32
		z64[:63] + "2" + // length = 2
		"6869" + strings.Repeat("0", 60) // "hi" right-padded to 32 bytes
	if got != want {
		t.Errorf("dynamic string:\n got=%s\nwant=%s", got, want)
	}
}

func TestEncodeABI_MixedStaticDynamic(t *testing.T) {
	// g(uint256,string,uint256) with (1, "a", 2): the string's offset accounts
	// for all three head words (3*32 = 0x60).
	raw := accounts.EncodeABI(accounts.Uint(big.NewInt(1)), accounts.StringArg("a"), accounts.Uint(big.NewInt(2)))
	want := "" +
		z64[:63] + "1" + // arg0 = 1
		z64[:62] + "60" + // arg1 offset = 96
		z64[:63] + "2" + // arg2 = 2
		z64[:63] + "1" + // string length = 1
		"61" + strings.Repeat("0", 62) // "a" padded
	if got := hexOf(raw); got != want {
		t.Errorf("mixed encode:\n got=%s\nwant=%s", got, want)
	}
}

func TestEncodeABI_NestedBytesPayload(t *testing.T) {
	// A governance-style proof: bytes = EncodeABI(address, uint256, string),
	// then wrapped as proposeMint(bytes). Just assert it round-trips in length and
	// the outer offset points past the single head word.
	proof := accounts.EncodeABI(accounts.Address("0x00000000000000000000000000000000000000ab"),
		accounts.Uint(big.NewInt(5)), accounts.StringArg("memo"))
	data := accounts.EncodeCallArgs("proposeMint(bytes)", accounts.Bytes(proof))
	body := strings.TrimPrefix(data, "0x")[8:] // drop selector
	if body[:64] != z64[:62]+"20" {
		t.Errorf("bytes arg offset should be 0x20, got %s", body[:64])
	}
	// length word of the bytes == len(proof)
	gotLen := new(big.Int)
	gotLen.SetString(body[64:128], 16)
	if int(gotLen.Int64()) != len(proof) {
		t.Errorf("bytes length = %d, want %d", gotLen, len(proof))
	}
}

func TestWordAt(t *testing.T) {
	// A two-word blob: word[0] = 1, word[1] = 0xab. WordAt is the inverse of the
	// head-word packing EncodeABI does.
	blob := "0x" + z64[:63] + "1" + z64[:62] + "ab"
	if w, ok := accounts.WordAt(blob, 0); !ok || w != "0x"+z64[:63]+"1" {
		t.Errorf("WordAt(blob,0) = %q,%v want word[0]", w, ok)
	}
	if w, ok := accounts.WordAt(blob, 1); !ok || w != "0x"+z64[:62]+"ab" {
		t.Errorf("WordAt(blob,1) = %q,%v want word[1]", w, ok)
	}
	// out of range and negative index -> not found.
	if _, ok := accounts.WordAt(blob, 2); ok {
		t.Errorf("WordAt accepted an out-of-range index")
	}
	if _, ok := accounts.WordAt(blob, -1); ok {
		t.Errorf("WordAt accepted a negative index")
	}
	// round-trip: WordAt(EncodeABI(Uint(n))) recovers n.
	raw := accounts.EncodeABI(accounts.Uint(big.NewInt(0xbeef)))
	w, ok := accounts.WordAt("0x"+hexOf(raw), 0)
	if !ok {
		t.Fatalf("WordAt on a packed word returned not-ok")
	}
	if n, good := accounts.WordToBig(w); !good || n.Int64() != 0xbeef {
		t.Errorf("round-trip WordToBig(WordAt(pack(0xbeef))) = %v,%v", n, good)
	}
}

func sel4(sig string) string { return strings.TrimPrefix(accounts.Selector(sig), "0x") }
func hexOf(b []byte) string {
	var s strings.Builder
	for _, c := range b {
		s.WriteString(hexByte(c))
	}
	return s.String()
}
func hexByte(c byte) string {
	const h = "0123456789abcdef"
	return string([]byte{h[c>>4], h[c&0xf]})
}
