// This file ports the native-coin adapter approve + transferFrom flow (from
// tests/regression f-system-contracts f1-03), exercising the ERC-20 allowance
// path end to end: an owner approves a spender, the spender moves the owner's
// tokens to a recipient, and the recipient's native balance grows by the amount.
//
// # Test: token-transfer-from-moves-balance
//
// Intent:   after an owner approves a spender on the native-coin adapter, the
//
//	spender can transferFrom(owner, recipient, amount) and the recipient's
//	balance increases by exactly amount.
//
// Applies:  stablenet. Requires "rpc".
// Method:   open the faucet wallet as the owner; pick a validator coinbase as the
//
//	spender (node-side signing); owner approves the spender for `amount`;
//	spender calls transferFrom(owner, recipient, amount); assert the
//	recipient's balance rose by `amount`.
//
// Pass:     the recipient's balance increases by exactly `amount`.
//
// This is chainbench TEST CODE (requirement #16): it drives real transactions
// (an owner-signed approve and a node-side-signed transferFrom), so it is only
// meaningful against a live multi-validator network (the sibling _test.go
// validates registration/gating).
package anzeon

import (
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// transferFromRecipient starts unfunded so its balance gain is exactly the
// transferred amount.
const transferFromRecipient = "0x00000000000000000000000000000000C0FFEE08"

func init() {
	testkit.Register(testkit.Case{
		Name:         "token-transfer-from-moves-balance",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           tokenTransferFromMovesBalance,
	})
}

func tokenTransferFromMovesBalance(t *testkit.T) {
	ctx := t.Ctx()
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	ownerWallet, err := ap.OpenWallet(ctx, key, primary.RPCURL)
	t.NoErr(err, "open owner wallet")
	owner := ownerWallet.Address()

	// The spender is a validator coinbase (unlocked, node-side signing), distinct
	// from the owner.
	vals := discoverValidators(t)
	var spender validator
	for _, v := range vals {
		if !strings.EqualFold(v.addr, owner) {
			spender = v
			break
		}
	}
	t.Truef(spender.client != nil, "need a validator distinct from the owner to act as spender")

	amount := big.NewInt(1_000_000_000_000_000) // 0.001 coin

	// Owner approves the spender for `amount` on the adapter.
	approveData := accounts.EncodeCall("approve(address,uint256)",
		accounts.AddressArg(spender.addr), amount.Bytes())
	approveCall, err := hex.DecodeString(strings.TrimPrefix(approveData, "0x"))
	t.NoErr(err, "decode approve calldata")
	approveHash, err := ownerWallet.Execute(ctx, nativeCoinAdapter, approveCall, nil)
	t.NoErr(err, "approve execute")

	c := rpc.Dial(primary.RPCURL)
	t.WaitFor(func() bool { return receiptSucceeded(ctx, c, approveHash) },
		90*time.Second, time.Second, "approve receipt")

	balBefore, err := c.BalanceAt(ctx, transferFromRecipient)
	t.NoErr(err, "recipient balance before")

	// Spender moves the owner's tokens to the recipient.
	tfData := accounts.EncodeCallArgs("transferFrom(address,address,uint256)",
		accounts.Address(owner), accounts.Address(transferFromRecipient), accounts.Uint(amount))
	tfHash, err := spender.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: spender.addr, To: nativeCoinAdapter, Data: tfData, Gas: govGas,
	})
	t.NoErr(err, "transferFrom")
	t.WaitFor(func() bool { return receiptSucceeded(ctx, spender.client, tfHash) },
		90*time.Second, time.Second, "transferFrom receipt")

	// The recipient's balance rose by exactly the transferred amount.
	t.WaitFor(func() bool {
		balAfter, err := c.BalanceAt(ctx, transferFromRecipient)
		return err == nil && new(big.Int).Sub(balAfter, balBefore).Cmp(amount) == 0
	}, 90*time.Second, time.Second, "recipient balance increased by the transferred amount")
}
