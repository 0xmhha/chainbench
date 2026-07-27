// This file ports the regular-account gas-tip forcing case (from tests/regression
// c-anzeon c-01). Anzeon forces a regular (non-authorized) account's transaction
// tip down to the header GasTip, regardless of how high a priority fee the sender
// requests — so the effective tip charged is the header GasTip, not the requested
// one.
//
// # Test: regular-account-gastip-forced
//
// Intent:   a regular account that requests a tip far above the header GasTip is
//
//	still charged only the header GasTip (anzeon forces it down).
//
// Applies:  stablenet. Requires "rpc".
// Method:   read the header GasTip (istanbul_getWbftExtraInfo); send a type-0x02
//
//	transfer from the faucet (a regular account) with an explicit tip 5x the
//	header GasTip; from the receipt take effectiveGasPrice and subtract the
//	inclusion block's baseFeePerGas to get the tip actually charged.
//
// Pass:     the tip charged equals the header GasTip (not the requested high tip).
//
// This is chainbench TEST CODE (requirement #16): it drives a real transaction,
// so the sibling _test.go validates registration/gating.
package anzeon

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// gastipRecipient receives the probe transfer (regression c-01 TEST_ACC_B).
const gastipRecipient = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"

func init() {
	testkit.Register(testkit.Case{
		Name:         "regular-account-gastip-forced",
		Category:     "gas-policy",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           regularAccountGastipForced,
	})
}

func regularAccountGastipForced(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	var latest string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_blockNumber", &latest), "eth_blockNumber")
	headerTip := headerGasTip(t, latest)
	t.Truef(headerTip.Sign() > 0, "header GasTip is positive (got %s)", headerTip)
	baseFee := latestBaseFee(t)

	// Request a tip 5x the header tip; a regular account must have it forced back
	// down to the header GasTip. feeCap is generous so the tx is includable.
	highTip := new(big.Int).Mul(headerTip, big.NewInt(5))
	feeCap := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), highTip)

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	hash, err := w.SendDynamicFeeGas(t.Ctx(), gastipRecipient, big.NewInt(1), feeCap, highTip)
	t.NoErr(err, "send high-tip dynamic-fee tx")

	tipUsed := effectiveTip(t, hash)
	t.Truef(tipUsed.Cmp(headerTip) == 0,
		"regular account tip forced to header GasTip %s (charged %s; requested high tip was %s)",
		headerTip, tipUsed, highTip)
}

// effectiveTip waits for the tx receipt and returns effectiveGasPrice minus the
// inclusion block's own baseFee — the priority fee actually charged. Using the
// inclusion block's baseFee (not a separately-read one) makes the charged tip
// exact, independent of any baseFee drift between blocks.
func effectiveTip(t *testkit.T, hash string) *big.Int {
	var rcpt struct {
		Status            string `json:"status"`
		EffectiveGasPrice string `json:"effectiveGasPrice"`
		BlockNumber       string `json:"blockNumber"`
	}
	t.WaitFor(func() bool {
		var raw json.RawMessage
		if err := t.Primary().Call(t.Ctx(), "eth_getTransactionReceipt", &raw, hash); err != nil {
			return false
		}
		if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
			return false
		}
		return json.Unmarshal(raw, &rcpt) == nil && rcpt.Status == "0x1"
	}, 90*time.Second, time.Second, "tx receipt with status 0x1")

	var block struct {
		BaseFeePerGas string `json:"baseFeePerGas"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &block, rcpt.BlockNumber, false),
		"eth_getBlockByNumber(inclusion)")
	egp := hexBig(t, rcpt.EffectiveGasPrice, "effectiveGasPrice")
	inclusionBase := hexBig(t, block.BaseFeePerGas, "baseFeePerGas")
	return new(big.Int).Sub(egp, inclusionBase)
}
