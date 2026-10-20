package accounts

import (
	"context"
	"math/big"
	"testing"
)

func TestSpecifiedLegacyRejectsInvalidInputBeforeRPC(t *testing.T) {
	for _, sample := range []struct {
		name   string
		change func(*LegacyTxArgs)
	}{
		{"nil value", func(a *LegacyTxArgs) { a.Value = nil }},
		{"nil price", func(a *LegacyTxArgs) { a.GasPrice = nil }},
		{"negative value", func(a *LegacyTxArgs) { a.Value = big.NewInt(-1) }},
		{"negative price", func(a *LegacyTxArgs) { a.GasPrice = big.NewInt(-1) }},
		{"bad recipient", func(a *LegacyTxArgs) { a.ToHex = "invalid" }},
		{"bad list address", func(a *LegacyTxArgs) { a.AccessList = []AccessTuple{{Address: "invalid"}} }},
		{"short storage key", func(a *LegacyTxArgs) { a.AccessList = []AccessTuple{{Address: a.ToHex, StorageKeys: []string{"0x01"}}} }},
	} {
		t.Run(sample.name, func(t *testing.T) {
			args := LegacyTxArgs{ToHex: "0x000000000000000000000000000000000000dead", Value: big.NewInt(1), GasPrice: big.NewInt(3), Gas: 70000}
			sample.change(&args)
			hash, err := (sdkWallet{}).SendLegacyTx(context.Background(), args)
			if err == nil || hash != "" {
				t.Fatalf("invalid input reached RPC or produced hash: %q %v", hash, err)
			}
		})
	}
}

func TestDynamicAccessListRejectsInvalidTupleBeforeRPC(t *testing.T) {
	for _, tuple := range []AccessTuple{
		{Address: "invalid"},
		{Address: "0x000000000000000000000000000000000000dead", StorageKeys: []string{"0x01"}},
	} {
		hash, err := (sdkWallet{}).SendDynamicFeeTx(context.Background(), DynamicTxArgs{Value: big.NewInt(1), GasFeeCap: big.NewInt(3), GasTipCap: big.NewInt(1), Gas: 70000, AccessList: []AccessTuple{tuple}})
		if err == nil || hash != "" {
			t.Fatalf("invalid tuple reached RPC or produced a hash: %q %v", hash, err)
		}
	}
}
