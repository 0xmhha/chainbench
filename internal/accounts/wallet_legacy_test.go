package accounts

import (
	"context"
	"math/big"
	"strings"
	"testing"
)

// TestSendLegacy_Validation covers the input checks that happen before any RPC.
func TestSendLegacy_Validation(t *testing.T) {
	w := sdkWallet{} // w is nil; validation runs first
	ctx := context.Background()
	const addr = "0x0000000000000000000000000000000000000001"

	if _, err := w.SendLegacy(ctx, "not-an-address", big.NewInt(1)); err == nil ||
		!strings.Contains(err.Error(), "recipient") {
		t.Errorf("bad recipient should error: %v", err)
	}
	if _, err := w.SendLegacy(ctx, addr, nil); err == nil ||
		!strings.Contains(err.Error(), "amount") {
		t.Errorf("nil amount should error: %v", err)
	}
}
