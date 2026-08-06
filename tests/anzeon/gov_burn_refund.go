// This file ports the GovMinter burn-refund lifecycle cases (from regression
// h-hardfork h-06, h-09, h-11), on the Boho v2 GovMinter via node-side signing.
//
// # Test: burn-cancel-refundable
//
// Intent:   cancelling a burn proposal makes the burned value refundable to the
//
//	proposer (refundableBalance > 0).
//
// Applies:  stablenet. Requires "rpc".
// Method:   proposeBurn(proof) with value from validator1; extract the proposalId;
//
//	cancelProposal(id) from validator1; read refundableBalance(validator1).
//
// Pass:     refundableBalance(proposer) > 0 after the cancel.
//
// # Test: burn-execute-no-refundable
//
// Intent:   an executed burn leaves no refundable balance.
//
// Applies:  stablenet. Requires "rpc".
// Method:   proposeBurn from validator1; approve to quorum (executing); read
//
//	refundableBalance(validator1).
//
// Pass:     refundableBalance(proposer) == 0 after execution.
//
// # Test: claim-zero-refund-reverts
//
// Intent:   claimBurnRefund from an account with no refundable balance is
//
//	rejected (submission error) or reverts (receipt status 0x0).
//
// Applies:  stablenet. Requires "rpc".
// Method:   claimBurnRefund() from a validator that never burned.
// Pass:     the call is rejected or mines with status 0x0.
//
// These are chainbench TEST CODE (requirement #16): live multi-validator
// governance flows (the sibling _test.go validates registration/gating).
package anzeon

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// burnRefundAmount is a modest payable burn value the proposer's coinbase covers.
var burnRefundAmount = big.NewInt(1_000_000_000_000_000) // 0.001 coin

func init() {
	testkit.Register(testkit.Case{
		Name:         "burn-cancel-refundable",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           burnCancelRefundable,
	})
	testkit.Register(testkit.Case{
		Name:         "burn-execute-no-refundable",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           burnExecuteNoRefundable,
	})
	testkit.Register(testkit.Case{
		Name:         "claim-zero-refund-reverts",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           claimZeroRefundReverts,
	})
	testkit.Register(testkit.Case{
		Name:         "burn-reject-refundable",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           burnRejectRefundable,
	})
	testkit.Register(testkit.Case{
		Name:         "claim-burn-refund-succeeds",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           claimBurnRefundSucceeds,
	})
	testkit.Register(testkit.Case{
		Name:         "claim-burn-refund-double-reverts",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           claimBurnRefundDoubleReverts,
	})
	testkit.Register(testkit.Case{
		Name:         "burn-refund-events",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           burnRefundEvents,
	})
}

// cancelForRefund proposes a burn from the proposer and cancels it, returning the
// cancel tx hash once the proposer's refundable balance is positive.
func cancelForRefund(t *testkit.T, proposer validator) string {
	ctx := t.Ctx()
	proposalID := proposeBurnFrom(t, proposer)
	cancelHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.CancelProposalCall(proposalID), Gas: govGas,
	})
	t.NoErr(err, "cancelProposal")
	t.WaitFor(func() bool { return receiptSucceeded(ctx, proposer.client, cancelHash) },
		60*time.Second, time.Second, "cancel receipt")
	t.WaitFor(func() bool { return refundableBalance(t, proposer.client, proposer.addr).Sign() > 0 },
		60*time.Second, time.Second, "refundableBalance > 0")
	return cancelHash
}

// claimRefund sends claimBurnRefund from v and returns the tx hash.
func claimRefund(t *testkit.T, v validator) string {
	h, err := v.client.SendTransaction(t.Ctx(), rpc.SendTxArgs{
		From: v.addr, To: govMinter, Data: govbind.ClaimBurnRefundCall(), Gas: govGas,
	})
	t.NoErr(err, "claimBurnRefund")
	return h
}

func burnRejectRefundable(t *testkit.T) {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]
	proposalID := proposeBurnFrom(t, proposer)

	// Disapprove from the other validators until the rejection quorum makes the
	// burned value refundable.
	for _, v := range vals[1:] {
		h, err := v.client.SendTransaction(ctx, rpc.SendTxArgs{
			From: v.addr, To: govMinter, Data: govbind.DisapproveProposalCall(proposalID), Gas: govGas,
		})
		t.NoErr(err, "disapproveProposal")
		t.WaitFor(func() bool { return receiptSucceeded(ctx, v.client, h) },
			60*time.Second, time.Second, "disapprove receipt")
		if refundableBalance(t, proposer.client, proposer.addr).Sign() > 0 {
			break
		}
	}
	t.WaitFor(func() bool { return refundableBalance(t, proposer.client, proposer.addr).Sign() > 0 },
		60*time.Second, time.Second, "refundableBalance > 0 after rejection")
}

func claimBurnRefundSucceeds(t *testkit.T) {
	ctx := t.Ctx()
	proposer := discoverValidators(t)[0]
	cancelForRefund(t, proposer)
	claimHash := claimRefund(t, proposer)
	t.WaitFor(func() bool { return receiptSucceeded(ctx, proposer.client, claimHash) },
		60*time.Second, time.Second, "claim receipt")
	// the refundable balance is consumed by the claim.
	t.WaitFor(func() bool { return refundableBalance(t, proposer.client, proposer.addr).Sign() == 0 },
		60*time.Second, time.Second, "refundable consumed after claim")
}

func claimBurnRefundDoubleReverts(t *testkit.T) {
	ctx := t.Ctx()
	proposer := discoverValidators(t)[0]
	cancelForRefund(t, proposer)

	first := claimRefund(t, proposer)
	t.WaitFor(func() bool { return receiptSucceeded(ctx, proposer.client, first) },
		60*time.Second, time.Second, "first claim succeeds")

	// The second claim has nothing to claim and must be rejected or revert.
	second, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.ClaimBurnRefundCall(), Gas: govGas,
	})
	if err != nil {
		return // rejected at submission — expected
	}
	t.WaitFor(func() bool { return receiptReverted(ctx, proposer.client, second) },
		60*time.Second, time.Second, "second claim reverts (status 0x0)")
}

func burnRefundEvents(t *testkit.T) {
	ctx := t.Ctx()
	proposer := discoverValidators(t)[0]

	// The cancel emits BurnDepositRefunded.
	cancelHash := cancelForRefund(t, proposer)
	t.WaitFor(func() bool {
		return receiptHasTopic(ctx, proposer.client, cancelHash, govbind.BurnDepositRefundedTopic)
	},
		60*time.Second, time.Second, "cancel receipt has BurnDepositRefunded")

	// The claim emits BurnRefundClaimed.
	claimHash := claimRefund(t, proposer)
	t.WaitFor(func() bool { return receiptHasTopic(ctx, proposer.client, claimHash, govbind.BurnRefundClaimedTopic) },
		60*time.Second, time.Second, "claim receipt has BurnRefundClaimed")
}

// receiptReverted reports whether a mined tx has status 0x0.
func receiptReverted(ctx context.Context, c *rpc.Client, hash string) bool {
	raw, err := c.TxReceipt(ctx, hash)
	if err != nil || len(raw) == 0 {
		return false
	}
	var r struct {
		Status string `json:"status"`
	}
	return json.Unmarshal(raw, &r) == nil && r.Status == "0x0"
}

// receiptHasTopic reports whether a successful tx's logs include a log with topic0.
func receiptHasTopic(ctx context.Context, c *rpc.Client, hash, topic0 string) bool {
	logs, ok := receiptLogs(ctx, c, hash)
	if !ok {
		return false
	}
	_, found := accounts.FindLog(logs, topic0)
	return found
}

// proposeBurnFrom proposes a burn (raw proof bytes; the Boho v2 GovMinter accepts
// arbitrary proofData) with the payable value from the proposer, returning the
// proposalId.
func proposeBurnFrom(t *testkit.T, proposer validator) *big.Int {
	proof, _ := hex.DecodeString("deadbeef")
	proposeHash, err := proposer.client.SendTransaction(t.Ctx(), rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.ProposeBurnCall(proof),
		Gas: govGas, Value: "0x" + burnRefundAmount.Text(16),
	})
	t.NoErr(err, "proposeBurn")
	return extractProposalID(t, proposer, proposeHash)
}

func refundableBalance(t *testkit.T, c *rpc.Client, addr string) *big.Int {
	bal, err := accounts.ReadUint(t.Ctx(), c.EthCall, govMinter, "refundableBalance(address)", accounts.AddressArg(addr))
	t.NoErr(err, "refundableBalance")
	return bal
}

func burnCancelRefundable(t *testkit.T) {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]
	proposalID := proposeBurnFrom(t, proposer)

	cancelHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.CancelProposalCall(proposalID), Gas: govGas,
	})
	t.NoErr(err, "cancelProposal")
	t.WaitFor(func() bool { return receiptSucceeded(ctx, proposer.client, cancelHash) },
		60*time.Second, time.Second, "cancel receipt")

	t.WaitFor(func() bool { return refundableBalance(t, proposer.client, proposer.addr).Sign() > 0 },
		60*time.Second, time.Second, "refundableBalance > 0 after cancel")
}

func burnExecuteNoRefundable(t *testkit.T) {
	vals := discoverValidators(t)
	proposer := vals[0]
	proposalID := proposeBurnFrom(t, proposer)
	approveToQuorum(t, govMinter, proposalID, proposer, vals[1:])

	bal := refundableBalance(t, proposer.client, proposer.addr)
	t.Truef(bal.Sign() == 0, "executed burn leaves 0 refundable (got %s)", bal)
}

func claimZeroRefundReverts(t *testkit.T) {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	// A validator that never burned has no refundable balance; claiming must be
	// rejected at submission or mine as a revert.
	claimer := vals[len(vals)-1]
	hash, err := claimer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: claimer.addr, To: govMinter, Data: govbind.ClaimBurnRefundCall(), Gas: govGas,
	})
	if err != nil {
		return // rejected at submission — the expected outcome
	}
	t.WaitFor(func() bool { return receiptReverted(ctx, claimer.client, hash) },
		60*time.Second, time.Second, "claim with zero refundable reverts (status 0x0)")
}
