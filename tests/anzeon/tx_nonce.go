// This file ports the nonce-ordering and replacement cases (from regression
// a-ethereum a2-04, a2-09). Both drive SendDynamicFeeTx with explicit nonces from
// a fresh funded account.
//
// # Test: nonce-ordering (a2-04)
//
// Intent:   transactions submitted out of nonce order are mined in ascending
//
//	nonce order once the gap is filled.
//
// Applies:  stablenet. Requires "rpc".
// Method:   from a fresh account at nonce N, submit N+2, N+1, N (reverse order);
//
//	assert all three mine (the account nonce reaches N+3).
//
// # Test: replacement-tx (a2-09)
//
// Intent:   a second transaction at the same nonce with a higher fee replaces the
//
//	first.
//
// Applies:  stablenet. Requires "rpc".
// Method:   submit a gapped tx at nonce N+1, then a replacement at N+1 with a
//
//	higher fee cap, then fill the gap at nonce N; assert the replacement mines
//	and the original does not.
//
// These are chainbench TEST CODE (requirement #16): live transactions, so the
// sibling _test.go validates registration/gating.
package anzeon

import (
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	reg := func(name string, fn func(*testkit.T)) {
		testkit.Register(testkit.Case{
			Name:         name,
			Category:     "accounts",
			ChainCompat:  []string{"stablenet"},
			RequiresCaps: []string{"rpc"},
			Fn:           fn,
		})
	}
	reg("nonce-ordering", nonceOrdering)
	reg("replacement-tx", replacementTx)
}

// readNonce returns the account's latest transaction count.
func readNonce(t *testkit.T, addr string) uint64 {
	var hexN string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getTransactionCount", &hexN, addr, "latest"), "eth_getTransactionCount")
	v, ok := new(big.Int).SetString(strings.TrimPrefix(hexN, "0x"), 16)
	if !ok {
		return 0
	}
	return v.Uint64()
}

// txMined reports whether the transaction has a receipt (is on chain).
func txMined(t *testkit.T, hash string) bool {
	var raw json.RawMessage
	if err := t.Primary().Call(t.Ctx(), "eth_getTransactionReceipt", &raw, hash); err != nil {
		return false
	}
	return len(raw) > 0 && strings.TrimSpace(string(raw)) != "null"
}

func nonceOrdering(t *testkit.T) {
	w := newFundedWallet(t)
	addr := w.Address()
	n := readNonce(t, addr)
	feeCap, tipCap := validFees(t)

	send := func(nonce uint64) {
		_, err := w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
			ToHex: gastipRecipient, Value: big.NewInt(1), Gas: 21000,
			GasFeeCap: feeCap, GasTipCap: tipCap, Nonce: &nonce,
		})
		t.NoErr(err, "send nonce")
	}

	// Submit in reverse order; N+2 and N+1 queue until N fills the gap.
	send(n + 2)
	send(n + 1)
	send(n)

	// All three mine in ascending nonce order — the account nonce reaches N+3.
	t.WaitFor(func() bool { return readNonce(t, addr) == n+3 },
		90*time.Second, time.Second, "all three out-of-order nonces mine")
}

func replacementTx(t *testkit.T) {
	w := newFundedWallet(t)
	addr := w.Address()
	n := readNonce(t, addr)
	feeCap, tipCap := validFees(t)
	gap := n + 1

	// Gapped tx at N+1 (waits for N).
	tx1, err := w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
		ToHex: gastipRecipient, Value: big.NewInt(1), Gas: 21000,
		GasFeeCap: feeCap, GasTipCap: tipCap, Nonce: &gap,
	})
	t.NoErr(err, "gapped tx1")

	// Replacement at the same nonce with a 20% higher fee (clears the 10% bump
	// requirement even after integer rounding).
	bump := func(x *big.Int) *big.Int {
		return new(big.Int).Div(new(big.Int).Mul(x, big.NewInt(12)), big.NewInt(10))
	}
	tx2, err := w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
		ToHex: gastipRecipient, Value: big.NewInt(1), Gas: 21000,
		GasFeeCap: bump(feeCap), GasTipCap: bump(tipCap), Nonce: &gap,
	})
	t.NoErr(err, "replacement tx2")

	// Fill the gap at nonce N so N+1 becomes mineable.
	fill := n
	_, err = w.SendDynamicFeeTx(t.Ctx(), accounts.DynamicTxArgs{
		ToHex: gastipRecipient, Value: big.NewInt(1), Gas: 21000,
		GasFeeCap: feeCap, GasTipCap: tipCap, Nonce: &fill,
	})
	t.NoErr(err, "gap-filling tx3")

	// The replacement mines; the original was dropped.
	t.WaitFor(func() bool { return txMined(t, tx2) }, 90*time.Second, time.Second, "replacement tx2 mined")
	t.Truef(!txMined(t, tx1), "the replaced tx1 must not be mined")
}
