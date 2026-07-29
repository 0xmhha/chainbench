// This file ports wemix4 TX-014 / TX-015: a 0x16 fee-delegated transfer carries
// two signatures (sender + fee payer); corrupting either one must make the node
// reject the transaction. The case builds a properly-formed tx, corrupts one
// signature via accounts.EncodeFeeDelegatedTampered, and submits the raw bytes
// with eth_sendRawTransaction, expecting an error.
//
//   - fee-delegated-sender-sig-invalid-rejected   (TX-014): corrupt the sender
//     signature -> rejected.
//   - fee-delegated-feepayer-sig-invalid-rejected (TX-015): corrupt the fee-payer
//     signature -> rejected.
//
// These are chainbench TEST CODE (requirement #16): live sends, so they are only
// meaningful against a running network (the sibling _test.go validates
// registration and foreign-chain gating).
package accounts

import (
	"encoding/hex"
	"math/big"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
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
	reg("fee-delegated-sender-sig-invalid-rejected", func(t *testkit.T) { rejectTamperedFD(t, "sender") })
	reg("fee-delegated-feepayer-sig-invalid-rejected", func(t *testkit.T) { rejectTamperedFD(t, "feepayer") })
}

func rejectTamperedFD(t *testkit.T, which string) {
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	if !ap.SupportsTxType(0x16) {
		t.Skip("chain %s does not support fee delegation (0x16)", t.NodeSet().Chain)
	}

	faucetKey, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, _ := openFaucetWallet(t)

	cid, err := t.Primary().ChainID(t.Ctx())
	t.NoErr(err, "eth_chainId")
	nonce := rejNonce(t, w.Address())
	feeCap, tipCap := rejValidFees(t)

	// The faucet is both sender and fee payer (a valid 0x16 shape); only the named
	// signature is corrupted.
	raw, err := accounts.EncodeFeeDelegatedTampered(faucetKey, faucetKey, txRejRecipient, big.NewInt(1),
		int64(cid), nonce, feeCap, tipCap, which)
	t.NoErr(err, "build tampered fee-delegated tx")

	var h string
	sendErr := t.Primary().Call(t.Ctx(), "eth_sendRawTransaction", &h, "0x"+hex.EncodeToString(raw))
	t.Truef(sendErr != nil, "a fee-delegated tx with an invalid %s signature must be rejected (got hash %q)", which, h)
}
