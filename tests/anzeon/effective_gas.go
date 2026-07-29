// This file ports the regular-account effective-gas-price case (from
// regression h-hardfork h-43).
//
// # Test: effective-gas-price-regular
//
// Intent:   a regular (non-authorized) account pays the full EIP-1559 price, so
//
//	its receipt effectiveGasPrice equals the inclusion block's baseFeePerGas
//	plus the suggested priority fee (anzeon charges the tip for regular
//	accounts).
//
// Applies:  stablenet. Requires "rpc".
// Method:   read the suggested tip; SendCoin from the faucet (a regular account);
//
//	from the receipt take effectiveGasPrice and the inclusion block; assert
//	effectiveGasPrice == that block's baseFeePerGas + tip.
//
// Pass:     effectiveGasPrice == inclusion baseFee + tip.
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
	"github.com/0xmhha/chainbench/pkg/testkit"
)

const egpRegularRecipient = "0x00000000000000000000000000000000C0FFEE11"

func init() {
	testkit.Register(testkit.Case{
		Name:         "effective-gas-price-regular",
		Category:     "gas-policy",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           effectiveGasPriceRegular,
	})
}

func effectiveGasPriceRegular(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	_, tip := baseFeeAndTip(t)

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	hash, err := w.SendCoin(t.Ctx(), egpRegularRecipient, big.NewInt(1))
	t.NoErr(err, "send")

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
	}, 90*time.Second, time.Second, "transfer receipt with status 0x1")

	// The base fee of the block the tx was included in.
	var block struct {
		BaseFeePerGas string `json:"baseFeePerGas"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &block, rcpt.BlockNumber, false),
		"eth_getBlockByNumber(inclusion)")

	egp := hexBig(t, rcpt.EffectiveGasPrice, "effectiveGasPrice")
	baseFee := hexBig(t, block.BaseFeePerGas, "baseFeePerGas")
	want := new(big.Int).Add(baseFee, tip)
	t.Truef(egp.Cmp(want) == 0, "effectiveGasPrice %s == inclusion baseFee %s + tip %s", egp, baseFee, tip)
}

// hexBig parses a 0x-hex quantity, failing the test on a malformed value.
func hexBig(t *testkit.T, hexQty, what string) *big.Int {
	v, ok := new(big.Int).SetString(strings.TrimPrefix(hexQty, "0x"), 16)
	t.Truef(ok, "%s %q is hex", what, hexQty)
	return v
}
