package accounts_test

import (
	"math/big"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

// TestEncodeCallTransferGolden pins the calldata the migrated
// estimate-gas-token-transfer spec (tests/specs/gas-policy) embeds, so the
// spec and the legacy case provably encode the same call.
func TestEncodeCallTransferGolden(t *testing.T) {
	got := accounts.EncodeCall("transfer(address,uint256)",
		accounts.AddressArg("0x00000000000000000000000000000000C0FFEE09"), big.NewInt(1000).Bytes())
	const want = "0xa9059cbb00000000000000000000000000000000000000000000000000000000c0ffee0900000000000000000000000000000000000000000000000000000000000003e8"
	if got != want {
		t.Fatalf("calldata:\n got %s\nwant %s", got, want)
	}
}
