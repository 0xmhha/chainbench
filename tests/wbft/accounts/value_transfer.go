// # Test: value-transfer
//
// Intent:   exercise the accounts SDK transaction path end to end — sign a
//
//	value transfer with a genesis-funded key, submit it, and confirm the
//	recipient's on-chain balance reflects the transfer.
//
// Applies:  stablenet (its preset alloc funds the faucet key below).
// Requires: the "rpc" capability.
// Method:   OpenWallet(faucetKey) -> SendCoin(recipient, amount) via the
//
//	stablenet account provider, then poll eth_getBalance until the
//	recipient holds exactly the sent amount.
//
// Pass:     recipient balance == amount within the wait window.
//
// This is chainbench TEST CODE (requirement #16). Unlike the read-only cases it
// drives a real transaction, so it is only meaningful against a live network
// (the sibling _test.go validates registration/gating, not the tx itself).
package accounts

import (
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

const (
	// faucetKeyHex is a genesis-funded key present in the stablenet preset
	// alloc (keys/preset/metadata.json). TEST FIXTURE ONLY.
	faucetKeyHex = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
	// transferRecipient starts unfunded so its post-transfer balance equals
	// exactly the amount sent.
	transferRecipient = "0x00000000000000000000000000000000C0FFEE01"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "value-transfer",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           valueTransfer,
	})
}

func valueTransfer(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")

	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")

	amount := big.NewInt(1_000_000_000_000_000_000) // 1 coin (1e18 wei)
	_, err = ap.Faucet(t.Ctx(), key, transferRecipient, amount, primary.RPCURL)
	t.NoErr(err, "faucet transfer")

	// The submitted transaction always mines eventually; inclusion is usually
	// one block but can lag right after launch on wbft, when verify reports
	// production before the validator mesh has fully converged and gossiped the
	// transaction to the current proposer. Allow a generous window so the case
	// is not flaky on that startup tail.
	t.WaitFor(func() bool {
		return balanceOf(t, transferRecipient).Cmp(amount) == 0
	}, 90*time.Second, time.Second, "recipient balance to equal the transferred amount")
}

// balanceOf reads an account's latest balance via the primary node; a failed
// or unparsable read yields zero so WaitFor keeps polling.
func balanceOf(t *testkit.T, addr string) *big.Int {
	var hexBal string
	if err := t.Primary().Call(t.Ctx(), "eth_getBalance", &hexBal, addr, "latest"); err != nil {
		return big.NewInt(0)
	}
	v, ok := new(big.Int).SetString(strings.TrimPrefix(hexBal, "0x"), 16)
	if !ok {
		return big.NewInt(0)
	}
	return v
}
