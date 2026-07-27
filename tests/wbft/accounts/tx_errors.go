// This file ports two transaction/call error-path checks from the legacy bash
// regression suite: insufficient-funds rejection
// (tests/regression/a-ethereum/a2-06-insufficient-funds) and eth_call revert
// surfacing (tests/regression/a-ethereum/a3-05-eth-call-revert). Both assert
// that the node returns an error rather than silently accepting the request.
//
// # Test: insufficient-funds-rejected
//
// Intent:   a value transfer that exceeds the sender's balance is rejected at
//
//	submission time rather than mined.
//
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   read the funded sender's balance, then attempt to send balance + 1
//
//	coin; the SDK submit (eth_sendRawTransaction) must return an error.
//
// Pass:     SendCoin returns a non-nil error.
//
// # Test: eth-call-revert-returns-error
//
// Intent:   an eth_call to a function that reverts returns a JSON-RPC error, not
//
//	a normal result.
//
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   deploy a Reverter contract whose fail() always reverts, wait for the
//
//	code to appear, then eth_call fail() and assert the call errors.
//
// Pass:     eth_call fail() returns a non-nil error.
//
// These are chainbench TEST CODE (requirement #16): they drive real
// transactions/calls against a live node (the sibling _test.go validates
// registration/gating).
package accounts

import (
	"encoding/hex"
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// insufficientRecipient receives the (rejected) over-balance transfer.
const insufficientRecipient = "0x00000000000000000000000000000000C0FFEE07"

// reverterInit is the creation bytecode of a Reverter contract whose fail()
// (selector 0xa9cc4718) always reverts with "BAD_INPUT" (solc 0.8.30,
// --optimize --no-cbor-metadata). Copied verbatim from the bash port.
const reverterInit = "6080604052348015600e575f5ffd5b50606a80601a5f395ff3fe6080604052348015600e575f5ffd5b50600436106026575f3560e01c8063a9cc471814602a575b5f5ffd5b60306032565b005b60405162461bcd60e51b815260206004820152600960248201526810905117d25394155560ba1b604482015260640160405180910390fd"

// reverterFailSelector is the 4-byte selector of fail().
const reverterFailSelector = "0xa9cc4718"

func init() {
	testkit.Register(testkit.Case{
		Name:         "insufficient-funds-rejected",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           insufficientFundsRejected,
	})
	testkit.Register(testkit.Case{
		Name:         "eth-call-revert-returns-error",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           ethCallRevertReturnsError,
	})
}

func insufficientFundsRejected(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	// Attempt to send strictly more than the sender holds: balance + 1 coin.
	bal := balanceOf(t, w.Address())
	amount := new(big.Int).Add(bal, big.NewInt(1_000_000_000_000_000_000))

	_, err = w.SendCoin(t.Ctx(), insufficientRecipient, amount)
	t.Truef(err != nil, "over-balance transfer of %s wei (balance %s) must be rejected", amount, bal)
}

func ethCallRevertReturnsError(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	initCode, err := hex.DecodeString(reverterInit)
	t.NoErr(err, "decode reverter init code")
	_, addr, err := w.Deploy(t.Ctx(), initCode, nil)
	t.NoErr(err, "deploy reverter")

	c := rpc.Dial(primary.RPCURL)
	t.WaitFor(func() bool {
		code, err := c.CodeAt(t.Ctx(), addr)
		return err == nil && code != "" && code != "0x"
	}, 90*time.Second, time.Second, "reverter contract to deploy")

	// fail() always reverts, so eth_call must surface a JSON-RPC error.
	_, err = c.EthCall(t.Ctx(), addr, reverterFailSelector)
	t.Truef(err != nil, "eth_call fail() must return an error for a reverting call")
}
