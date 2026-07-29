// # Test: legacy-transfer
//
// Intent:   exercise the type 0x00 (EIP-155 legacy) transaction path and confirm
//
//	the node reports it as a legacy tx. Ported from the legacy bash
//	regression regression/a-ethereum/a2-01-legacy-tx.
//
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   open a funded wallet and SendLegacy to a fresh recipient; assert the
//
//	transaction reports type 0x0 via eth_getTransactionByHash, then poll the
//	recipient's balance.
//
// Pass:     tx type == 0x0 and recipient balance == amount within the window.
//
// This is chainbench TEST CODE (requirement #16): it drives a real transaction,
// so it is only meaningful against a live network (the sibling _test.go
// validates registration/gating).
package accounts

import (
	"encoding/hex"
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// legacyRecipient starts unfunded so its post-transfer balance equals exactly
// the amount sent.
const legacyRecipient = "0x00000000000000000000000000000000C0FFEE03"

func init() {
	testkit.Register(testkit.Case{
		Name:         "legacy-transfer",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           legacyTransfer,
	})
}

func legacyTransfer(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")

	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")

	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	amount := big.NewInt(1_000_000_000_000_000_000) // 1 coin (1e18 wei)
	txHash, err := w.SendLegacy(t.Ctx(), legacyRecipient, amount)
	t.NoErr(err, "legacy transfer")

	// The node must report it as a legacy (0x0) transaction.
	var tx struct {
		Type string `json:"type"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getTransactionByHash", &tx, txHash), "eth_getTransactionByHash")
	t.Equalf(tx.Type, "0x0", "transaction type is legacy (0x0)")

	t.WaitFor(func() bool {
		return balanceOf(t, legacyRecipient).Cmp(amount) == 0
	}, 90*time.Second, time.Second, "recipient balance to equal the legacy-transferred amount")
}
