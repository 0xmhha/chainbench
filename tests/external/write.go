// Package external holds chain-agnostic write cases (empty ChainCompat) that run
// against ANY chain — a first-party chain or a project-supplied one attached via
// `chainbench setup --manifest` / `chainbench test --rpc`. They act as a
// funded account whose key the operator supplies out of band
// (CHAINBENCH_FUNDED_KEY), so no key is committed and the cases are not tied to a
// specific chain's preset. They Skip when no funded key is configured.
//
// This completes the hybrid external-manifest model's write side and covers the
// z-layer2 send-tx (RT-Z-02) and fee-delegation (RT-Z-05) scenarios on an
// arbitrary L2.
//
// # Test: external-value-transfer
//
// Intent:   the funded account can transfer value on the attached chain.
// Applies:  all chains. Requires "rpc" + a configured funded key.
// Method:   SendCoin to a fresh recipient; poll the recipient balance.
//
// # Test: external-fee-delegated-transfer
//
// Intent:   the funded account can send a 0x16 fee-delegated transfer.
// Applies:  all chains that support tx type 0x16. Requires "rpc" + a funded key.
// Method:   SendFeeDelegated (sender == fee payer == funded key) to a fresh
//
//	recipient; poll the recipient balance.
//
// These are chainbench TEST CODE (requirement #16): live transactions, so the
// sibling _test.go validates registration and the skip-without-key gating.
package external

import (
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// xferAmount is a small value transfer (1e15 wei) that leaves headroom for gas.
const xferAmount = 1_000_000_000_000_000

func init() {
	testkit.Register(testkit.Case{
		Name:         "external-value-transfer",
		Category:     "accounts",
		RequiresCaps: []string{"rpc"},
		Fn:           externalValueTransfer,
	})
	testkit.Register(testkit.Case{
		Name:         "external-fee-delegated-transfer",
		Category:     "accounts",
		RequiresCaps: []string{"rpc"},
		Fn:           externalFeeDelegatedTransfer,
	})
}

// requireFundedKey returns the configured funded-account key, or skips the case
// when none is set (chain-agnostic write cases need an operator-supplied key).
func requireFundedKey(t *testkit.T) []byte {
	key, ok := t.FundedKey()
	if !ok {
		t.Skip("no funded key configured (set CHAINBENCH_FUNDED_KEY to run write cases on this chain)")
	}
	return key
}

func openFunded(t *testkit.T, key []byte) accounts.Wallet {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open funded wallet")
	return w
}

func freshRecipient(t *testkit.T) string {
	_, addr, err := accounts.GenerateKey()
	t.NoErr(err, "generate recipient")
	return addr
}

func balanceOf(t *testkit.T, addr string) *big.Int {
	var hexBal string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBalance", &hexBal, addr, "latest"), "eth_getBalance")
	v, ok := new(big.Int).SetString(strings.TrimPrefix(hexBal, "0x"), 16)
	if !ok {
		return big.NewInt(0)
	}
	return v
}

func externalValueTransfer(t *testkit.T) {
	key := requireFundedKey(t)
	w := openFunded(t, key)
	to := freshRecipient(t)

	_, err := w.SendCoin(t.Ctx(), to, big.NewInt(xferAmount))
	t.NoErr(err, "send value")
	t.WaitFor(func() bool { return balanceOf(t, to).Cmp(big.NewInt(xferAmount)) == 0 },
		90*time.Second, time.Second, "recipient balance to equal the amount sent")
}

func externalFeeDelegatedTransfer(t *testkit.T) {
	key := requireFundedKey(t)
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	if !ap.SupportsTxType(0x16) {
		t.Skip("chain %s does not support fee-delegation (0x16)", t.NodeSet().Chain)
	}
	w := openFunded(t, key)
	to := freshRecipient(t)

	// sender == fee payer == the funded key: a valid 0x16 tx that exercises the
	// dual-signature encode and the chain's acceptance of the type.
	_, err = w.SendFeeDelegated(t.Ctx(), key, to, big.NewInt(xferAmount))
	t.NoErr(err, "fee-delegated transfer")
	t.WaitFor(func() bool { return balanceOf(t, to).Cmp(big.NewInt(xferAmount)) == 0 },
		90*time.Second, time.Second, "recipient balance to equal the amount sent")
}
