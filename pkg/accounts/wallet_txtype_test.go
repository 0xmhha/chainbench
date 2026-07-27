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

func TestSendExplicitGas_Validation(t *testing.T) {
	w := sdkWallet{}
	ctx := context.Background()
	const addr = "0x0000000000000000000000000000000000000001"
	one := big.NewInt(1)

	// bad recipient on each explicit-gas method.
	if _, err := w.SendDynamicFeeGas(ctx, "bad", one, one, one); err == nil || !strings.Contains(err.Error(), "recipient") {
		t.Errorf("SendDynamicFeeGas bad recipient: %v", err)
	}
	if _, err := w.SendLegacyGas(ctx, "bad", one, one); err == nil || !strings.Contains(err.Error(), "recipient") {
		t.Errorf("SendLegacyGas bad recipient: %v", err)
	}
	if _, err := w.SendAccessListGas(ctx, "bad", one, one); err == nil || !strings.Contains(err.Error(), "recipient") {
		t.Errorf("SendAccessListGas bad recipient: %v", err)
	}
	// nil fee / amount.
	if _, err := w.SendDynamicFeeGas(ctx, addr, one, nil, one); err == nil || !strings.Contains(err.Error(), "fee") {
		t.Errorf("SendDynamicFeeGas nil feecap: %v", err)
	}
	if _, err := w.SendLegacyGas(ctx, addr, one, nil); err == nil || !strings.Contains(err.Error(), "gas price") {
		t.Errorf("SendLegacyGas nil gasprice: %v", err)
	}
}
