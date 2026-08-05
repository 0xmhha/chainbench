// This file adds a token-metadata read case (ported from f-system-contracts),
// exercising dynamic-string return decoding.
//
// # Test: token-metadata
//
// Intent:   the native-coin adapter answers the ERC-20 name()/symbol() reads,
//
//	which return dynamic strings — the go-stablenet token is "WKRC".
//
// Applies:  stablenet. Requires: the "rpc" capability.
// Method:   eth_call name() and symbol(); decode the dynamic string returns and
//
//	assert both equal "WKRC" (the genesis native-coin-adapter params).
//
// Pass:     name() == symbol() == "WKRC".
//
// This is chainbench TEST CODE (requirement #16): run by the testrun phase
// against a live NodeSet (the sibling _test.go runs it against a mock).
package anzeon

import (
	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "token-metadata",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           tokenMetadata,
	})
}

func tokenMetadata(t *testkit.T) {
	call := caller(t)
	name, err := accounts.ReadString(t.Ctx(), call, nativeCoinAdapter, "name()")
	t.NoErr(err, "name()")
	t.Equalf(name, "WKRC", "token name")
	symbol, err := accounts.ReadString(t.Ctx(), call, nativeCoinAdapter, "symbol()")
	t.NoErr(err, "symbol()")
	t.Equalf(symbol, "WKRC", "token symbol")
}
