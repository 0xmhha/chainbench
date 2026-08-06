// This file ports two transaction-rejection cases (from regression
// a-ethereum a2-05a and d-fee-delegation d-05). Both submit a transaction the
// node must refuse: an underpriced tip, and a fee-delegated transfer whose fee
// payer cannot cover the gas.
//
// (a2-05b — gasFeeCap below the minimum — is already covered by
// feecap-below-min-rejected in gas_boundary.go.)
//
// # Test: tipcap-underpriced-rejected (a2-05a)
//
// Intent:   a type-0x02 tx with tipCap below MinTip is rejected (ErrUnderpriced).
// Applies:  stablenet. Requires "rpc".
// Method:   send with a valid feeCap (baseFee + header GasTip) but tipCap = 1 wei;
//
//	expect a submission error.
//
// # Test: feepayer-insufficient-rejected (d-05)
//
// Intent:   a 0x16 fee-delegated transfer whose fee payer has no balance is
//
//	rejected.
//
// Applies:  stablenet. Requires "rpc".
// Method:   the faucet transfers value while a FRESH (unfunded) account is the fee
//
//	payer; expect a submission error.
//
// These are chainbench TEST CODE (requirement #16): live sends, so the sibling
// _test.go validates registration/gating.
package anzeon

import (
	"encoding/hex"
	"math/big"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "tipcap-underpriced-rejected",
		Category:     "gas-policy",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           tipcapUnderpricedRejected,
	})
	testkit.Register(testkit.Case{
		Name:         "feepayer-insufficient-rejected",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           feepayerInsufficientRejected,
	})
}

// openFaucetWallet opens the genesis-funded faucet wallet on the primary node.
func openFaucetWallet(t *testkit.T) accounts.Wallet {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open faucet wallet")
	return w
}

func tipcapUnderpricedRejected(t *testkit.T) {
	headerTip := headerGasTip(t, latestBlockHex(t))
	baseFee := latestBaseFee(t)
	// A valid feeCap (>= baseFee + tip) with a tipCap of 1 wei, which is below
	// MinTip — the node must reject it as underpriced.
	feeCap := new(big.Int).Add(baseFee, headerTip)
	w := openFaucetWallet(t)
	_, err := w.SendDynamicFeeGas(t.Ctx(), gastipRecipient, big.NewInt(1), feeCap, big.NewInt(1))
	t.Truef(err != nil, "a tipCap below MinTip must be rejected (got nil error)")
}

func feepayerInsufficientRejected(t *testkit.T) {
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	t.Truef(ap.SupportsTxType(0x16), "chain %s must support fee-delegation (0x16)", t.NodeSet().Chain)

	// A fresh, unfunded fee payer.
	feeKey, _, err := accounts.GenerateKey()
	t.NoErr(err, "generate fee-payer key")

	// The faucet (funded sender) transfers value while the unfunded account pays
	// the gas — the node must reject it (the fee payer cannot cover the gas).
	faucet := openFaucetWallet(t)
	_, err = faucet.SendFeeDelegated(t.Ctx(), feeKey, gastipRecipient, big.NewInt(1))
	t.Truef(err != nil, "a fee-delegated tx with an unfunded fee payer must be rejected (got nil error)")
}
