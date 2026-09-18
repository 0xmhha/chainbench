package accounts_test

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

func TestDecodeString(t *testing.T) {
	// ABI encoding of the string "WKRC": offset 0x20, length 4, "WKRC" padded.
	enc := strings.Repeat("0", 62) + "20" + // offset = 32
		strings.Repeat("0", 63) + "4" + // length = 4
		"574b524300000000000000000000000000000000000000000000000000000000" // "WKRC"
	if got := accounts.DecodeString("0x" + enc); got != "WKRC" {
		t.Errorf("DecodeString = %q, want WKRC", got)
	}
	// round-trip against the encoder.
	rt := accounts.EncodeABI(accounts.StringArg("hello world"))
	if got := accounts.DecodeString("0x" + hexString(rt)); got != "hello world" {
		t.Errorf("round-trip = %q", got)
	}
	// malformed inputs yield "".
	for _, bad := range []string{"0x", "0xzz", "0x1234"} {
		if got := accounts.DecodeString(bad); got != "" {
			t.Errorf("malformed %q decoded to %q", bad, got)
		}
	}
}

func TestReadString(t *testing.T) {
	call := func(_ context.Context, _, _ string) (string, error) {
		enc := strings.Repeat("0", 62) + "20" + strings.Repeat("0", 63) + "3" +
			"57454d000000000000000000000000000000000000000000000000000000000" + "0" // "WEM"
		return "0x" + enc, nil
	}
	got, err := accounts.ReadString(context.Background(), call, "0xc0", "symbol()")
	if err != nil || got != "WEM" {
		t.Errorf("ReadString = %q (%v)", got, err)
	}
}

func hexString(b []byte) string {
	const h = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, h[c>>4], h[c&0xf])
	}
	return string(out)
}
