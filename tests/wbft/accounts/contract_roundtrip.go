// # Test: contract-roundtrip
//
// Intent:   exercise the accounts SDK contract path end to end — deploy
//
//	creation bytecode with a genesis-funded key, then read the deployed
//	contract back over eth_call.
//
// Applies:  stablenet, wbft (both EVM, wbft consensus family).
// Requires: the "rpc" capability.
// Method:   Deploy a minimal contract whose runtime returns the constant 42,
//
//	then poll eth_call until the deployed code answers 42.
//
// Pass:     eth_call on the deployed address returns 0x..2a within the window.
//
// This is chainbench TEST CODE (requirement #16); it drives real transactions,
// so it is only meaningful against a live network (the sibling _test.go
// validates registration/gating, not the deployment itself).
package accounts

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// returns42Init is the creation bytecode of a contract whose runtime code
// (602a60005260206000f3) returns the 32-byte value 42 for any call.
const returns42Init = "600a600c600039600a6000f3602a60005260206000f3"

func init() {
	testkit.Register(testkit.Case{
		Name:         "contract-roundtrip",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           contractRoundtrip,
	})
}

func contractRoundtrip(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")

	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")

	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	initCode, err := hex.DecodeString(returns42Init)
	t.NoErr(err, "decode init code")

	_, addr, err := w.Deploy(t.Ctx(), initCode, nil)
	t.NoErr(err, "deploy contract")

	// Wait for the deployment to mine and the contract to answer 42.
	c := rpc.Dial(primary.RPCURL)
	t.WaitFor(func() bool {
		res, err := c.EthCall(t.Ctx(), addr, "0x")
		return err == nil && strings.HasSuffix(res, "2a")
	}, 90*time.Second, time.Second, "deployed contract to return 42 over eth_call")
}
