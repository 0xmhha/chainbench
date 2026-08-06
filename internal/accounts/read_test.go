package accounts_test

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

func TestReadUint(t *testing.T) {
	ctx := context.Background()

	// a caller that echoes the calldata selector back as the result count.
	var gotTo, gotData string
	call := func(_ context.Context, to, data string) (string, error) {
		gotTo, gotData = to, data
		return "0x" + strings.Repeat("0", 62) + "2a", nil // 42
	}
	v, err := accounts.ReadUint(ctx, call, "0xc0", "balanceOf(address)", accounts.AddressArg("0x00000000000000000000000000000000000000ab"))
	if err != nil || v.Int64() != 42 {
		t.Fatalf("ReadUint = %v (%v)", v, err)
	}
	if gotTo != "0xc0" || !strings.HasPrefix(gotData, "0x70a08231") {
		t.Errorf("call args: to=%s data=%s", gotTo, gotData)
	}

	// an undecodable result is an error.
	bad := func(_ context.Context, _, _ string) (string, error) { return "0xnothex", nil }
	if _, err := accounts.ReadUint(ctx, bad, "0x", "totalSupply()"); err == nil {
		t.Error("undecodable result should error")
	}
}
