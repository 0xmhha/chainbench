// This file adds a token approve/allowance write case (ported from
// tests/regression f-system-contracts f1-03 approve-transferFrom), exercising a
// state-changing contract call plus the resulting read and event.
//
// # Test: token-approve-sets-allowance
//
// Intent:   approving a spender on the native-coin adapter must set the
//
//	on-chain allowance and emit an Approval(owner, spender, value) event.
//
// Applies:  stablenet. Requires: the "rpc" capability.
// Method:   Execute approve(spender, amount) on the adapter; assert the receipt
//
//	carries an Approval event to the spender with the amount, then read
//	allowance(owner, spender) and assert it equals the amount.
//
// Pass:     the Approval event matches and allowance(owner, spender) == amount.
//
// This is chainbench TEST CODE (requirement #16): it drives a real transaction,
// so it is only meaningful against a live network (the sibling _test.go validates
// registration/gating).
package anzeon

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// approveSpender is a fixed spender address; the allowance amount is arbitrary.
const approveSpender = "0x00000000000000000000000000000000C0FFEE05"

func init() {
	testkit.Register(testkit.Case{
		Name:         "token-approve-sets-allowance",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           tokenApproveSetsAllowance,
	})
}

func tokenApproveSetsAllowance(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")

	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")
	owner := w.Address()

	amount := big.NewInt(100)
	data := accounts.EncodeCall("approve(address,uint256)",
		accounts.AddressArg(approveSpender), amount.Bytes())
	callData, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	t.NoErr(err, "decode calldata")

	execHash, err := w.Execute(t.Ctx(), nativeCoinAdapter, callData, nil)
	t.NoErr(err, "approve execute")

	approvalTopic := accounts.EventTopic("Approval(address,address,uint256)")
	c := rpc.Dial(primary.RPCURL)
	t.WaitFor(func() bool {
		raw, err := c.TxReceipt(t.Ctx(), execHash)
		if err != nil || len(raw) == 0 {
			return false
		}
		var r struct {
			Status string         `json:"status"`
			Logs   []accounts.Log `json:"logs"`
		}
		if err := json.Unmarshal(raw, &r); err != nil || r.Status != "0x1" {
			return false
		}
		log, found := accounts.FindLog(r.Logs, approvalTopic)
		if !found || len(log.Topics) < 3 {
			return false
		}
		// Approval(owner indexed, spender indexed, value): topics[2] = spender.
		return strings.EqualFold(accounts.TopicToAddress(log.Topics[2]), approveSpender)
	}, 90*time.Second, time.Second, "Approval event for the spender")

	// The on-chain allowance must now equal the approved amount.
	got, err := accounts.ReadUint(t.Ctx(), c.EthCall, nativeCoinAdapter,
		"allowance(address,address)", accounts.AddressArg(owner), accounts.AddressArg(approveSpender))
	t.NoErr(err, "allowance read")
	t.Equalf(got.String(), amount.String(), "allowance(owner, spender) equals the approved amount")
}
