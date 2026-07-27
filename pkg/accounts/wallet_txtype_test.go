package accounts

import (
	"context"
	"math/big"
	"strings"
	"testing"
)

// TestSendDynamicFee_Validation and TestSendAccessList_Validation cover the input
// checks that run before any RPC (a nil wallet suffices — validation is first).
func TestSendDynamicFee_Validation(t *testing.T) {
	w := sdkWallet{}
	ctx := context.Background()
	const addr = "0x0000000000000000000000000000000000000001"

	if _, err := w.SendDynamicFee(ctx, "not-an-address", big.NewInt(1)); err == nil ||
		!strings.Contains(err.Error(), "recipient") {
		t.Errorf("bad recipient should error: %v", err)
	}
	if _, err := w.SendDynamicFee(ctx, addr, nil); err == nil ||
		!strings.Contains(err.Error(), "amount") {
		t.Errorf("nil amount should error: %v", err)
	}
}

func TestSendAccessList_Validation(t *testing.T) {
	w := sdkWallet{}
	ctx := context.Background()
	const addr = "0x0000000000000000000000000000000000000001"

	if _, err := w.SendAccessList(ctx, "not-an-address", big.NewInt(1)); err == nil ||
		!strings.Contains(err.Error(), "recipient") {
		t.Errorf("bad recipient should error: %v", err)
	}
	if _, err := w.SendAccessList(ctx, addr, nil); err == nil ||
		!strings.Contains(err.Error(), "amount") {
		t.Errorf("nil amount should error: %v", err)
	}
}
