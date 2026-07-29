// This file ports the gas-limit and receipt-status cases (from regression
// a-ethereum a2-07, a3-06, a3-07). They drive the low-level SendDynamicFeeTx with
// an explicit gas limit so a tx that reverts or runs out of gas still mines (the
// higher-level helpers auto-estimate and would fail at submission instead).
//
// # Test: gaslimit-exceeded-rejected (a2-07)
//
// Intent:   a tx whose gas limit exceeds the block gas limit is rejected.
// Applies:  stablenet. Requires "rpc".
// Method:   send with gas = blockGasLimit + 1; expect a submission error.
//
// # Test: revert-tx-status-zero (a3-06)
//
// Intent:   a call that reverts mines with receipt.status == 0x0.
// Applies:  stablenet. Requires "rpc".
// Method:   deploy a contract whose runtime always REVERTs, call it with an
//
//	explicit gas limit, and assert the receipt status is 0x0.
//
// # Test: out-of-gas-consumes-all (a3-07)
//
// Intent:   a call that runs out of gas mines with status 0x0 and gasUsed == the
//
//	provided gas limit.
//
// Applies:  stablenet. Requires "rpc".
// Method:   deploy a contract whose runtime loops forever, call it with a fixed
//
//	gas limit, and assert status 0x0 and gasUsed == gasLimit.
//
// These are chainbench TEST CODE (requirement #16): live transactions, so the
// sibling _test.go validates registration/gating.
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

// Minimal EVM runtimes wrapped in a deployer that returns them:
//
//	revert: PUSH1 0 PUSH1 0 REVERT              (always reverts)
//	loop:   JUMPDEST PUSH1 0 JUMP               (infinite loop, consumes all gas)
const (
	revertInitCode = "600580600b6000396000f360006000fd"
	loopInitCode   = "600480600b6000396000f35b600056"
)

func init() {
	reg := func(name string, fn func(*testkit.T)) {
		testkit.Register(testkit.Case{
			Name:         name,
			Category:     "gas-policy",
			ChainCompat:  []string{"stablenet"},
			RequiresCaps: []string{"rpc"},
			Fn:           fn,
		})
	}
	reg("gaslimit-exceeded-rejected", gaslimitExceededRejected)
	reg("revert-tx-status-zero", revertTxStatusZero)
	reg("out-of-gas-consumes-all", outOfGasConsumesAll)
}

// validFees returns a fee cap and tip that comfortably clear the anzeon minimum.
func validFees(t *testkit.T) (feeCap, tipCap *big.Int) {
	tip := headerGasTip(t, latestBlockHex(t))
	base := latestBaseFee(t)
	return new(big.Int).Add(new(big.Int).Mul(base, big.NewInt(2)), tip), tip
}

// deployRuntime deploys initHex and returns the contract address once its code
// is on chain.
func deployRuntime(t *testkit.T, w accounts.Wallet, initHex string) string {
	code, err := hex.DecodeString(initHex)
	t.NoErr(err, "decode init code")
	_, contract, err := w.Deploy(t.Ctx(), code, nil)
	t.NoErr(err, "deploy contract")
	t.WaitFor(func() bool {
		var got string
		return t.Primary().Call(t.Ctx(), "eth_getCode", &got, contract, "latest") == nil && got != "" && got != "0x"
	}, 60*time.Second, time.Second, "deployed contract has code")
	return contract
}

// waitReceipt waits for the tx receipt and returns its status and gasUsed.
func waitReceipt(t *testkit.T, hash string) (status string, gasUsed *big.Int) {
	primary, _ := t.NodeSet().Primary()
	c := rpc.Dial(primary.RPCURL)
	var rcpt struct {
		Status  string `json:"status"`
		GasUsed string `json:"gasUsed"`
	}
	t.WaitFor(func() bool {
		var raw json.RawMessage
		if err := c.Call(t.Ctx(), "eth_getTransactionReceipt", &raw, hash); err != nil {
			return false
		}
		if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
			return false
		}
		return json.Unmarshal(raw, &rcpt) == nil && rcpt.Status != ""
	}, 60*time.Second, time.Second, "tx receipt")
	return rcpt.Status, hexBig(t, rcpt.GasUsed, "gasUsed")
}

func gaslimitExceededRejected(t *testkit.T) {
	var block struct {
		GasLimit string `json:"gasLimit"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &block, "latest", false), "eth_getBlockByNumber")
	blockGas := hexBig(t, block.GasLimit, "gasLimit")
	over := new(big.Int).Add(blockGas, big.NewInt(1)).Uint64()

	feeCap, tipCap := validFees(t)
	w := openFaucetWallet(t)
	_, err := w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
		ToHex: gastipRecipient, Value: big.NewInt(1), Gas: over, GasFeeCap: feeCap, GasTipCap: tipCap,
	})
	t.Truef(err != nil, "a tx with gas above the block gas limit must be rejected (got nil error)")
}

func revertTxStatusZero(t *testkit.T) {
	w := openFaucetWallet(t)
	contract := deployRuntime(t, w, revertInitCode)

	feeCap, tipCap := validFees(t)
	hash, err := w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
		ToHex: contract, Value: big.NewInt(0), Gas: 100000, GasFeeCap: feeCap, GasTipCap: tipCap,
	})
	t.NoErr(err, "call the revert contract")

	status, _ := waitReceipt(t, hash)
	t.Truef(status == "0x0", "a reverting call mines with status 0x0 (got %s)", status)
}

func outOfGasConsumesAll(t *testkit.T) {
	w := openFaucetWallet(t)
	contract := deployRuntime(t, w, loopInitCode)

	feeCap, tipCap := validFees(t)
	const gasLimit = uint64(50000)
	hash, err := w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
		ToHex: contract, Value: big.NewInt(0), Gas: gasLimit, GasFeeCap: feeCap, GasTipCap: tipCap,
	})
	t.NoErr(err, "call the loop contract")

	status, gasUsed := waitReceipt(t, hash)
	t.Truef(status == "0x0", "an out-of-gas call mines with status 0x0 (got %s)", status)
	t.Truef(gasUsed.Cmp(new(big.Int).SetUint64(gasLimit)) == 0,
		"out-of-gas consumes the full gas limit (gasUsed=%s limit=%d)", gasUsed, gasLimit)
}
