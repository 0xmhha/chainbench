package accounts

import (
	"context"
	"strings"
	"testing"
)

func TestSendSetCode_Validation(t *testing.T) {
	w := sdkWallet{} // w is nil; validation runs before any RPC
	ctx := context.Background()
	const delegate = "0x1111111111111111111111111111111111111111"

	if _, err := w.SendSetCode(ctx, []byte{1, 2, 3}, "not-an-address"); err == nil ||
		!strings.Contains(err.Error(), "delegate") {
		t.Errorf("bad delegate should error: %v", err)
	}
	if _, err := w.SendSetCode(ctx, []byte{}, delegate); err == nil ||
		!strings.Contains(err.Error(), "authority") {
		t.Errorf("bad authority key should error: %v", err)
	}
}

func TestGenerateKey(t *testing.T) {
	k1, a1, err := GenerateKey()
	if err != nil || len(k1) == 0 || !strings.HasPrefix(a1, "0x") || len(a1) != 42 {
		t.Fatalf("generate key 1: key=%d addr=%q err=%v", len(k1), a1, err)
	}
	k2, a2, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if a1 == a2 {
		t.Error("two generated keys share an address")
	}
	// the generated key derives back to its reported address.
	got, err := addressForKey(k1)
	if err != nil || !strings.EqualFold(got, a1) {
		t.Errorf("addressForKey(k1)=%s (err %v), want %s", got, err, a1)
	}
	_ = k2
}
