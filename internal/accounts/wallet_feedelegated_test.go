package accounts

import (
	"context"
	"math/big"
	"strings"
	"testing"
)

// TestSendFeeDelegated_Validation covers the input checks that happen before any
// RPC, so the SDK wallet (nil here) is never touched.
func TestSendFeeDelegated_Validation(t *testing.T) {
	w := sdkWallet{} // w is nil; validation runs first
	ctx := context.Background()
	const addr = "0x0000000000000000000000000000000000000001"

	if _, err := w.SendFeeDelegated(ctx, []byte{1, 2, 3}, "not-an-address", big.NewInt(1)); err == nil ||
		!strings.Contains(err.Error(), "recipient") {
		t.Errorf("bad recipient should error: %v", err)
	}
	if _, err := w.SendFeeDelegated(ctx, []byte{1, 2, 3}, addr, nil); err == nil ||
		!strings.Contains(err.Error(), "amount") {
		t.Errorf("nil amount should error: %v", err)
	}
	if _, err := w.SendFeeDelegated(ctx, []byte{}, addr, big.NewInt(1)); err == nil ||
		!strings.Contains(err.Error(), "fee-payer") {
		t.Errorf("bad fee-payer key should error: %v", err)
	}
}
