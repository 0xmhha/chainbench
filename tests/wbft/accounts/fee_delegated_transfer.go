// # Test: fee-delegated-transfer
//
// Intent:   exercise the 0x16 fee-delegation transaction path — a chainbench-
//
//	distinctive tx type where one account transfers value while a second
//	account pays the gas (dual signature). Ported from the legacy bash
//	regression suite regression/d-fee-delegation.
//
// Applies:  stablenet, wbft (the wbft family accepts tx type 0x16; the case also
//
//	guards on the provider's SupportsTxType(0x16)).
//
// Requires: the "rpc" capability.
// Method:   open a wallet for a genesis-funded key and SendFeeDelegated to a
//
//	fresh recipient, using the same funded key as the fee payer (sender ==
//	fee payer is a valid 0x16 tx and still exercises the dual-sign encode +
//	chain acceptance of the type). Then poll the recipient's balance.
//
// Pass:     recipient balance == amount within the wait window.
//
// This is chainbench TEST CODE (requirement #16): it drives a real transaction,
// so it is only meaningful against a live network (the sibling _test.go
// validates registration/gating, not the tx itself).
package accounts

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// feeDelegRecipient starts unfunded so its post-transfer balance equals exactly
// the amount sent.
const feeDelegRecipient = "0x00000000000000000000000000000000C0FFEE02"

func init() {
	testkit.Register(testkit.Case{
		Name:         "fee-delegated-transfer",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           feeDelegatedTransfer,
	})
}

func feeDelegatedTransfer(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	t.Truef(ap.SupportsTxType(0x16), "chain %s must support fee-delegation (0x16)", t.NodeSet().Chain)

	key := fundedKey(t)

	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	amount := big.NewInt(1_000_000_000_000_000_000) // 1 coin (1e18 wei)
	// sender == fee payer == the funded key: a valid 0x16 tx that still
	// exercises the dual-signature encode and the chain's acceptance of the type.
	_, err = w.SendFeeDelegated(t.Ctx(), key, feeDelegRecipient, amount)
	t.NoErr(err, "fee-delegated transfer")

	t.WaitFor(func() bool {
		return balanceOf(t, feeDelegRecipient).Cmp(amount) == 0
	}, 90*time.Second, time.Second, "recipient balance to equal the fee-delegated amount")
}
