// This file ports the blacklisted-fee-payer case (from regression
// e-blacklist-authorized e-03). In a 0x16 fee-delegation transaction the fee
// payer is a distinct account; if it is blacklisted the transaction must be
// rejected. The SDK's static value-transfer guard checks the sender and
// recipient but NOT the fee payer, so the transaction reaches the node, which
// enforces the fee-payer blacklist.
//
// Like the authorized cases, this authorizes/blacklists a FRESH generated
// account at runtime via governance (no preset "blacklisted" key needed): the
// bench funds an account, blacklists it through GovCouncil, and uses it as the
// fee payer.
//
// # Test: feepayer-blacklisted-rejected
//
// Intent:   a fee-delegated transfer whose fee payer is blacklisted is rejected.
//
// Applies:  stablenet. Requires "rpc".
// Method:   fund a fresh account and blacklist it via governance; have the faucet
//
//	(a non-blacklisted sender) send a 0x16 transfer to a fresh recipient with
//	the blacklisted account as fee payer.
//
// Pass:     the send is rejected with a "blacklist" error.
//
// This is chainbench TEST CODE (requirement #16): live governance + tx flow, so
// the sibling _test.go validates registration/gating.
package anzeon

import (
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// feepayerRecipient is an unrelated (non-blacklisted) recipient for the transfer.
const feepayerRecipient = "0x00000000000000000000000000000000C0FFEE13"

func init() {
	testkit.Register(testkit.Case{
		Name:         "feepayer-blacklisted-rejected",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           feepayerBlacklistedRejected,
	})
}

func feepayerBlacklistedRejected(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	t.Truef(ap.SupportsTxType(0x16), "chain %s must support fee-delegation (0x16)", t.NodeSet().Chain)

	// A fresh fee-payer: fund it first (so the rejection is about the blacklist,
	// not insufficient funds), then blacklist it via governance. Funding must
	// precede the blacklist — a transfer to an already-blacklisted recipient is
	// itself rejected.
	feeKey, feeAddr, err := accounts.GenerateKey()
	t.NoErr(err, "generate fee-payer key")

	fkey, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	faucet, err := ap.OpenWallet(t.Ctx(), fkey, primary.RPCURL)
	t.NoErr(err, "open faucet wallet")

	fundHash, err := faucet.SendCoin(t.Ctx(), feeAddr, tenEther())
	t.NoErr(err, "fund fee-payer")
	c := rpc.Dial(primary.RPCURL)
	t.WaitFor(func() bool { return receiptSucceeded(t.Ctx(), c, fundHash) },
		90*time.Second, time.Second, "fund receipt")

	blacklistAndWait(t, feeAddr)

	// The faucet (a non-blacklisted sender) transfers value while the blacklisted
	// account pays the gas. The static guard does not inspect the fee payer, so
	// the tx reaches the node, which rejects it for the blacklisted fee payer.
	_, err = faucet.SendFeeDelegated(t.Ctx(), feeKey, feepayerRecipient, big.NewInt(1))
	t.Truef(err != nil && strings.Contains(strings.ToLower(err.Error()), "blacklist"),
		"a fee-delegated tx with a blacklisted fee payer must be rejected (got %v)", err)
}
