// This file ports the fee-delegation invalid-signature cases (from regression
// d-fee-delegation d-03, d-04). A 0x16 fee-delegated transfer carries two
// signatures (sender + fee payer); if either is corrupted the node must reject
// the transaction. The bench builds a properly-formed tx, corrupts one signature
// with accounts.EncodeFeeDelegatedTampered, and submits the raw bytes.
//
// # Test: fd-sender-sig-invalid-rejected (d-03)
//
// Intent:   a fee-delegated tx with a corrupted sender signature is rejected.
// Applies:  stablenet. Requires "rpc".
// Method:   build a tx (faucet as sender and fee payer), corrupt the sender
//
//	signature, eth_sendRawTransaction; expect an error.
//
// # Test: fd-feepayer-sig-invalid-rejected (d-04)
//
// Intent:   a fee-delegated tx with a corrupted fee-payer signature is rejected.
// Applies:  stablenet. Requires "rpc".
// Method:   as above, corrupting the fee-payer signature.
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
	reg := func(name string, fn func(*testkit.T)) {
		testkit.Register(testkit.Case{
			Name:         name,
			Category:     "accounts",
			ChainCompat:  []string{"stablenet"},
			RequiresCaps: []string{"rpc"},
			Fn:           fn,
		})
	}
	reg("fd-sender-sig-invalid-rejected", func(t *testkit.T) { rejectTamperedFD(t, "sender") })
	reg("fd-feepayer-sig-invalid-rejected", func(t *testkit.T) { rejectTamperedFD(t, "feepayer") })
}

func rejectTamperedFD(t *testkit.T, which string) {
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	t.Truef(ap.SupportsTxType(0x16), "chain %s must support fee-delegation (0x16)", t.NodeSet().Chain)

	faucetKey := fundedKey(t)
	w := openFaucetWallet(t)

	cid, err := t.Primary().ChainID(t.Ctx())
	t.NoErr(err, "eth_chainId")
	nonce := readNonce(t, w.Address())
	feeCap, tipCap := validFees(t)

	// The faucet is both sender and fee payer (a valid 0x16 shape); only the named
	// signature is corrupted.
	raw, err := accounts.EncodeFeeDelegatedTampered(faucetKey, faucetKey, gastipRecipient, big.NewInt(1),
		int64(cid), nonce, feeCap, tipCap, which)
	t.NoErr(err, "build tampered fee-delegated tx")

	var h string
	sendErr := t.Primary().Call(t.Ctx(), "eth_sendRawTransaction", &h, "0x"+hex.EncodeToString(raw))
	t.Truef(sendErr != nil, "a fee-delegated tx with an invalid %s signature must be rejected (got hash %q)", which, h)
}
