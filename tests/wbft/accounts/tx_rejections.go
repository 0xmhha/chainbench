// This file ports the wemix4 transaction-policy cases that the testkit did not
// yet cover on the wbft chain: fee/gas rejections, nonce ordering, same-nonce
// replacement, and an unfunded fee payer. They complement the accepted-path tx
// cases (value-transfer, typed-tx, fee-delegated-transfer).
//
//   - dynamic-fee-below-basefee-rejected      (TX-002): a maxFeePerGas below the
//     base fee is rejected by the node.
//   - gas-limit-exceeds-block-rejected        (TX-012): a tx whose gas limit is
//     above the block gas limit is rejected.
//   - out-of-order-nonces-mine                (TX-010): nonces submitted N+2, N+1,
//     N all mine, in ascending order.
//   - same-nonce-replacement                  (TX-013): a second tx at the same
//     nonce with a higher fee replaces the first; only the replacement mines.
//   - fee-delegated-unfunded-feepayer-rejected (TX-016): a 0x16 fee-delegation tx
//     whose fee payer holds no balance is rejected.
//
// These are chainbench TEST CODE (requirement #16): they drive real
// transactions, so they are only meaningful against a live network (the sibling
// _test.go validates registration and foreign-chain gating).
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
	txRejRecipient = "0x00000000000000000000000000000000C0FFEE10"
	// unfundedFeePayerHex is a fixed TEST-ONLY key that is never funded in any
	// genesis; used as an insolvent fee payer.
	unfundedFeePayerHex = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func init() {
	reg := func(name string, fn func(*testkit.T)) {
		testkit.Register(testkit.Case{
			Name:         name,
			Category:     "accounts",
			ChainCompat:  []string{"wbft"},
			RequiresCaps: []string{"rpc"},
			Fn:           fn,
		})
	}
	reg("dynamic-fee-below-basefee-rejected", dynamicFeeBelowBasefeeRejected)
	reg("gas-limit-exceeds-block-rejected", gasLimitExceedsBlockRejected)
	reg("out-of-order-nonces-mine", outOfOrderNoncesMine)
	reg("same-nonce-replacement", sameNonceReplacement)
	reg("fee-delegated-unfunded-feepayer-rejected", feeDelegatedUnfundedFeepayerRejected)
}

// rejHexUint reads a 0x-hex quantity from an eth_ call and returns it as big.Int.
func rejHexUint(t *testkit.T, method string, params ...any) *big.Int {
	var s string
	t.NoErr(t.Primary().Call(t.Ctx(), method, &s, params...), method)
	v, ok := new(big.Int).SetString(strings.TrimPrefix(s, "0x"), 16)
	t.Truef(ok, "%s returned non-hex %q", method, s)
	return v
}

// rejValidFees returns a comfortably-valid (feeCap, tipCap): feeCap = 2*baseFee +
// tip, so ordinary sends are accepted.
func rejValidFees(t *testkit.T) (feeCap, tipCap *big.Int) {
	var blk struct {
		BaseFeePerGas string `json:"baseFeePerGas"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &blk, "latest", false), "eth_getBlockByNumber(latest)")
	base, ok := new(big.Int).SetString(strings.TrimPrefix(blk.BaseFeePerGas, "0x"), 16)
	t.Truef(ok, "baseFeePerGas non-hex %q", blk.BaseFeePerGas)
	tip := rejHexUint(t, "eth_maxPriorityFeePerGas")
	return new(big.Int).Add(new(big.Int).Mul(base, big.NewInt(2)), tip), tip
}

// rejNonce reads addr's latest nonce.
func rejNonce(t *testkit.T, addr string) uint64 {
	return rejHexUint(t, "eth_getTransactionCount", addr, "latest").Uint64()
}

// rejBlockGasLimit reads the latest block's gas limit.
func rejBlockGasLimit(t *testkit.T) uint64 {
	var blk struct {
		GasLimit string `json:"gasLimit"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &blk, "latest", false), "eth_getBlockByNumber(latest)")
	v, ok := new(big.Int).SetString(strings.TrimPrefix(blk.GasLimit, "0x"), 16)
	t.Truef(ok, "gasLimit non-hex %q", blk.GasLimit)
	return v.Uint64()
}

func dynamicFeeBelowBasefeeRejected(t *testkit.T) {
	w, _ := openFaucetWallet(t)
	// A 1-wei fee cap is far below any live base fee.
	_, err := w.SendDynamicFeeGas(t.Ctx(), txRejRecipient, big.NewInt(1), big.NewInt(1), big.NewInt(1))
	t.Truef(err != nil, "a maxFeePerGas below the base fee must be rejected")
}

func gasLimitExceedsBlockRejected(t *testkit.T) {
	w, _ := openFaucetWallet(t)
	feeCap, tipCap := rejValidFees(t)
	over := rejBlockGasLimit(t) + 1
	_, err := w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
		ToHex: txRejRecipient, Value: big.NewInt(1), Gas: over,
		GasFeeCap: feeCap, GasTipCap: tipCap,
	})
	t.Truef(err != nil, "a gas limit above the block gas limit (%d) must be rejected", over-1)
}

func outOfOrderNoncesMine(t *testkit.T) {
	w, _ := openFaucetWallet(t)
	addr := w.Address()
	n := rejNonce(t, addr)
	feeCap, tipCap := rejValidFees(t)

	send := func(nonce uint64) {
		_, err := w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
			ToHex: txRejRecipient, Value: big.NewInt(1), Gas: 21000,
			GasFeeCap: feeCap, GasTipCap: tipCap, Nonce: &nonce,
		})
		t.NoErr(err, "send out-of-order nonce")
	}
	// Submit in reverse; N+2 and N+1 queue until N fills the gap.
	send(n + 2)
	send(n + 1)
	send(n)

	t.WaitFor(func() bool { return rejNonce(t, addr) == n+3 },
		90*time.Second, time.Second, "all three out-of-order nonces to mine")
}

func sameNonceReplacement(t *testkit.T) {
	w, c := openFaucetWallet(t)
	addr := w.Address()
	n := rejNonce(t, addr)
	feeCap, tipCap := rejValidFees(t)
	gap := n + 1

	// Gapped tx at N+1 (waits for N).
	tx1, err := w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
		ToHex: txRejRecipient, Value: big.NewInt(1), Gas: 21000,
		GasFeeCap: feeCap, GasTipCap: tipCap, Nonce: &gap,
	})
	t.NoErr(err, "gapped tx1")

	// Replacement at the same nonce with a 20% higher fee (clears the 10% bump
	// minimum even after integer rounding).
	bump := func(x *big.Int) *big.Int {
		return new(big.Int).Div(new(big.Int).Mul(x, big.NewInt(12)), big.NewInt(10))
	}
	tx2, err := w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
		ToHex: txRejRecipient, Value: big.NewInt(1), Gas: 21000,
		GasFeeCap: bump(feeCap), GasTipCap: bump(tipCap), Nonce: &gap,
	})
	t.NoErr(err, "replacement tx2 at the same nonce")

	// Fill nonce N so N+1 becomes mineable.
	fill := n
	_, err = w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
		ToHex: txRejRecipient, Value: big.NewInt(1), Gas: 21000,
		GasFeeCap: feeCap, GasTipCap: tipCap, Nonce: &fill,
	})
	t.NoErr(err, "gap-filling tx3")

	mined := func(hash string) bool {
		raw, err := c.TxReceipt(t.Ctx(), hash)
		return err == nil && raw != nil
	}
	t.WaitFor(func() bool { return mined(tx2) }, 90*time.Second, time.Second, "replacement tx2 to mine")
	t.Truef(!mined(tx1), "the replaced tx1 must not mine")
}

func feeDelegatedUnfundedFeepayerRejected(t *testkit.T) {
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	if !ap.SupportsTxType(0x16) {
		t.Skip("chain %s does not support fee delegation (0x16)", t.NodeSet().Chain)
	}
	w, _ := openFaucetWallet(t)
	feePayer, err := hex.DecodeString(unfundedFeePayerHex)
	t.NoErr(err, "decode unfunded fee-payer key")

	// Sender (funded) is fine, but the fee payer holds nothing -> the node must
	// reject the tx (the fee payer cannot cover gas).
	_, err = w.SendFeeDelegated(t.Ctx(), feePayer, txRejRecipient, big.NewInt(1))
	t.Truef(err != nil, "a fee-delegation tx with an unfunded fee payer must be rejected")
}
