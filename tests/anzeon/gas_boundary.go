// This file ports the anzeon minimum-fee boundary cases (from regression
// h-hardfork h-20..h-24), exercising the explicit-gas wallet sends.
//
// # Test: feecap-exact-min-accepted
//
// Intent:   a DynamicFeeTx with gasFeeCap == baseFee + tip (the anzeon minimum)
//
//	is accepted and mines.
//
// # Test: feecap-above-min-accepted
//
// Intent:   a DynamicFeeTx with gasFeeCap above the minimum is accepted.
//
// # Test: feecap-below-min-rejected
//
// Intent:   a DynamicFeeTx with gasFeeCap below the minimum is rejected by the
//
//	node (submission error).
//
// # Test: legacy-gasprice-below-min-rejected / accesslist-gasprice-below-min-rejected
//
// Intent:   a legacy / access-list tx with gasPrice below the minimum is rejected.
//
// Applies:  stablenet (the anzeon min-fee policy). Requires "rpc".
//
// These are chainbench TEST CODE (requirement #16): they drive real transactions,
// so they are only meaningful against a live network (the sibling _test.go
// validates registration/gating).
package anzeon

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/testkit"
)

const gasBoundaryRecipient = "0x00000000000000000000000000000000C0FFEE0E"

func init() {
	reg := func(name string, fn func(*testkit.T)) {
		testkit.Register(testkit.Case{
			Name: name, Category: "gas-policy",
			ChainCompat: []string{"stablenet"}, RequiresCaps: []string{"rpc"}, Fn: fn,
		})
	}
	reg("feecap-exact-min-accepted", feecapExactMinAccepted)
	reg("feecap-above-min-accepted", feecapAboveMinAccepted)
	reg("feecap-below-min-rejected", feecapBelowMinRejected)
	reg("legacy-gasprice-below-min-rejected", legacyGaspriceBelowMinRejected)
	reg("accesslist-gasprice-below-min-rejected", accesslistGaspriceBelowMinRejected)
}

// gasBoundaryWallet opens the faucet wallet on the primary node.
func gasBoundaryWallet(t *testkit.T) accounts.Wallet {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")
	return w
}

// baseFeeAndTip reads the latest base fee and the suggested priority fee.
func baseFeeAndTip(t *testkit.T) (*big.Int, *big.Int) {
	bf := latestBaseFee(t)
	var tipHex string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_maxPriorityFeePerGas", &tipHex), "eth_maxPriorityFeePerGas")
	tip, ok := new(big.Int).SetString(strings.TrimPrefix(tipHex, "0x"), 16)
	t.Truef(ok, "priority fee %q is hex", tipHex)
	return bf, tip
}

// waitTxSuccess waits for hash to mine with receipt status 0x1 via the primary.
func waitTxSuccess(t *testkit.T, hash, what string) {
	t.WaitFor(func() bool {
		var raw json.RawMessage
		if err := t.Primary().Call(t.Ctx(), "eth_getTransactionReceipt", &raw, hash); err != nil {
			return false
		}
		if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
			return false
		}
		var r struct {
			Status string `json:"status"`
		}
		return json.Unmarshal(raw, &r) == nil && r.Status == "0x1"
	}, 90*time.Second, time.Second, what)
}

func feecapExactMinAccepted(t *testkit.T) {
	w := gasBoundaryWallet(t)
	bf, tip := baseFeeAndTip(t)
	feeCap := new(big.Int).Add(bf, tip) // exact minimum
	hash, err := w.SendDynamicFeeGas(t.Ctx(), gasBoundaryRecipient, big.NewInt(1), feeCap, tip)
	t.NoErr(err, "send exact-min dynamic-fee tx")
	waitTxSuccess(t, hash, "exact-min gasFeeCap tx mines")
}

func feecapAboveMinAccepted(t *testkit.T) {
	w := gasBoundaryWallet(t)
	bf, tip := baseFeeAndTip(t)
	// baseFee*2 + tip is comfortably above the minimum.
	feeCap := new(big.Int).Add(new(big.Int).Mul(bf, big.NewInt(2)), tip)
	hash, err := w.SendDynamicFeeGas(t.Ctx(), gasBoundaryRecipient, big.NewInt(1), feeCap, tip)
	t.NoErr(err, "send above-min dynamic-fee tx")
	waitTxSuccess(t, hash, "above-min gasFeeCap tx mines")
}

func feecapBelowMinRejected(t *testkit.T) {
	w := gasBoundaryWallet(t)
	// A gasFeeCap of 1 wei is far below the anzeon minimum base fee.
	_, err := w.SendDynamicFeeGas(t.Ctx(), gasBoundaryRecipient, big.NewInt(1), big.NewInt(1), big.NewInt(1))
	t.Truef(err != nil, "below-min gasFeeCap must be rejected")
}

func legacyGaspriceBelowMinRejected(t *testkit.T) {
	w := gasBoundaryWallet(t)
	_, err := w.SendLegacyGas(t.Ctx(), gasBoundaryRecipient, big.NewInt(1), big.NewInt(1))
	t.Truef(err != nil, "below-min legacy gasPrice must be rejected")
}

func accesslistGaspriceBelowMinRejected(t *testkit.T) {
	w := gasBoundaryWallet(t)
	_, err := w.SendAccessListGas(t.Ctx(), gasBoundaryRecipient, big.NewInt(1), big.NewInt(1))
	t.Truef(err != nil, "below-min access-list gasPrice must be rejected")
}
