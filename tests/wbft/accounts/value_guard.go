// This file ports the value-transfer guard cases (from regression
// e-blacklist-authorized e-05, e-06): the accounts SDK rejects a value transfer
// to the zero address or to a precompile before submitting it.
//
// # Test: zero-address-transfer-blocked
//
// Intent:   a value transfer to the zero address is rejected.
//
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   OpenWallet(faucetKey).SendCoin(0x0, 1); expect a rejection.
// Pass:     SendCoin returns a "zero address" error.
//
// # Test: precompile-transfer-blocked
//
// Intent:   a value transfer to a precompile address is rejected.
//
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   OpenWallet(faucetKey).SendCoin(0x..01, 1); expect a rejection.
// Pass:     SendCoin returns a "precompile" error.
//
// These are chainbench TEST CODE (requirement #16): they drive the accounts SDK
// send path (whose static value-transfer guard rejects these targets before any
// RPC), so the sibling _test.go validates registration/gating.
package accounts

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "zero-address-transfer-blocked",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           func(t *testkit.T) { guardRejects(t, "0x0000000000000000000000000000000000000000", "zero address") },
	})
	testkit.Register(testkit.Case{
		Name:         "precompile-transfer-blocked",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           func(t *testkit.T) { guardRejects(t, "0x0000000000000000000000000000000000000001", "precompile") },
	})
}

// guardRejects opens the faucet wallet and asserts SendCoin to `to` is rejected
// with an error containing want.
func guardRejects(t *testkit.T, to, want string) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	_, err = w.SendCoin(t.Ctx(), to, big.NewInt(1))
	t.Truef(err != nil && strings.Contains(strings.ToLower(err.Error()), want),
		"transfer to %s must be rejected with %q (got %v)", to, want, err)
}
