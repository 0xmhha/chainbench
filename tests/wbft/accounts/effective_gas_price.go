// This file ports the effective-gas-price receipt check (from tests/regression
// a-ethereum a2-08), over the accounts SDK transaction path.
//
// # Test: effective-gas-price
//
// Intent:   a mined transaction's receipt reports a positive effectiveGasPrice
//
//	and consumes at least the 21000 intrinsic gas of a transfer.
//
// Applies:  stablenet, wbft (the wbft family). Requires: the "rpc" capability.
// Method:   faucet a 1-wei transfer, wait for the receipt, and assert
//
//	status == 0x1, effectiveGasPrice > 0, and gasUsed >= 21000.
//
// Pass:     the receipt mines with a positive effective gas price and >= 21000 gas.
//
// This is chainbench TEST CODE (requirement #16): it drives a real transaction,
// so it is only meaningful against a live network (the sibling _test.go validates
// registration/gating).
package accounts

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// effGasRecipient is an arbitrary recipient for the fee-check transfer.
const effGasRecipient = "0x00000000000000000000000000000000C0FFEE0B"

func init() {
	testkit.Register(testkit.Case{
		Name:         "effective-gas-price",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           effectiveGasPrice,
	})
}

func effectiveGasPrice(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")

	hash, err := ap.Faucet(t.Ctx(), key, effGasRecipient, big.NewInt(1), primary.RPCURL)
	t.NoErr(err, "faucet transfer")

	var rcpt struct {
		Status            string `json:"status"`
		GasUsed           string `json:"gasUsed"`
		EffectiveGasPrice string `json:"effectiveGasPrice"`
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

	egp, ok := new(big.Int).SetString(strings.TrimPrefix(rcpt.EffectiveGasPrice, "0x"), 16)
	t.Truef(ok && egp.Sign() > 0, "effectiveGasPrice is positive (got %q)", rcpt.EffectiveGasPrice)
	gu, ok := new(big.Int).SetString(strings.TrimPrefix(rcpt.GasUsed, "0x"), 16)
	t.Truef(ok && gu.Cmp(big.NewInt(21000)) >= 0, "gasUsed >= 21000 (got %q)", rcpt.GasUsed)
}
