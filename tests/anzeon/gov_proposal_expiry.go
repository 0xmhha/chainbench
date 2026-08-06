// This file ports the proposal-expiry case (from regression
// f-system-contracts f3-06). A GovValidator proposal that is not executed before
// its expiry window can be transitioned to Expired via expireProposal, after
// which it can no longer execute.
//
// The default proposal expiry is 604800s (7 days), too long for a test, so this
// case runs only on a network launched with the short-expiry overlay
// (pkg/chains/stablenet/overlays/short-expiry.json, which sets GovValidator
// expiry=30s and advertises the "short-expiry" capability).
//
// # Test: proposal-expiry-transitions
//
// Intent:   a proposal past its expiry transitions to Expired.
// Applies:  stablenet. Requires "rpc" and "short-expiry".
// Method:   proposeGasTip (a distinct value, never executed), wait past the
//
//	expiry window, call expireProposal, and assert the proposal status is
//	Expired.
//
// This is chainbench TEST CODE (requirement #16): a live governance flow with a
// real-time wait, so the sibling _test.go validates registration/gating.
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "proposal-expiry-transitions",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc", "short-expiry"},
		Fn:           proposalExpiryTransitions,
	})
}

func proposalExpiryTransitions(t *testkit.T) {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]

	// Propose a gasTip change with a distinct value; it is never approved to
	// quorum, so it stays Voting and then expires (no gasTip change occurs).
	proposeData := accounts.EncodeCallArgs("proposeGasTip(uint256)", accounts.Uint(big.NewInt(30000000000001)))
	h, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govValidator, Data: proposeData, Gas: govGas,
	})
	t.NoErr(err, "proposeGasTip")
	id := extractProposalID(t, proposer, h)

	// Wait past the overlay-shortened expiry window (30s) with margin.
	select {
	case <-ctx.Done():
		return
	case <-time.After(35 * time.Second):
	}

	// expireProposal transitions the proposal to Expired.
	expHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govValidator,
		Data: accounts.EncodeCallArgs("expireProposal(uint256)", accounts.Uint(id)), Gas: govGas,
	})
	t.NoErr(err, "expireProposal")
	t.WaitFor(func() bool { return receiptSucceeded(ctx, proposer.client, expHash) },
		60*time.Second, time.Second, "expireProposal mined")

	t.WaitFor(func() bool {
		ret, err := proposer.client.EthCall(ctx, govValidator, govbind.ProposalsCall(id))
		if err != nil {
			return false
		}
		st, ok := govbind.DecodeProposalStatus(ret)
		return ok && st == govbind.ProposalExpired
	}, 30*time.Second, time.Second, "proposal transitions to Expired")
}
