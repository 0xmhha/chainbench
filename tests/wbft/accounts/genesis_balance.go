// Package accounts holds wbft-family account/funding test cases.
//
// # Test: genesis-balance
//
// Intent:   verify the genesis pre-funded (alloc) accounts are actually funded
//
//	on-chain — the prerequisite for any faucet/transaction scenario.
//
// Applies:  stablenet, wbft (the preset alloc funds the validator accounts).
// Requires: the "rpc" capability.
// Method:   query eth_getBalance for a known preset-funded address on the
//
//	primary node and assert a non-zero balance.
//
// Pass:     balance > 0 for the funded account.
//
// This is chainbench TEST CODE (requirement #16): it lives under tests/, is
// named tests/<family>/<category>/<name>.go, carries this godoc header, and
// registers a testkit.Case at init — it is executed by the testrun phase
// against a live NodeSet, not by `go test` (the sibling _test.go validates the
// registration/convention).
package accounts

import (
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "genesis-balance",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           genesisBalance,
	})
}

func genesisBalance(t *testkit.T) {
	// node1's validator account is pre-funded by the stablenet preset alloc.
	const funded = "0xc17d493883eaa3b4cceb0f214b273392d562f9d8"

	var hexBal string
	err := t.Primary().Call(t.Ctx(), "eth_getBalance", &hexBal, funded, "latest")
	t.NoErr(err, "eth_getBalance")

	bal, ok := new(big.Int).SetString(strings.TrimPrefix(hexBal, "0x"), 16)
	t.Truef(ok, "balance %q is not hex", hexBal)
	t.Truef(bal.Sign() > 0, "funded account must have non-zero balance, got %s", hexBal)
}
