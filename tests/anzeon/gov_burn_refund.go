// This file ports the GovMinter burn-refund lifecycle cases (from tests/regression
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
	"encoding/hex"
	"encoding/json"
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/testkit"
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
	t.WaitFor(func() bool {
		raw, err := claimer.client.TxReceipt(ctx, hash)
		if err != nil || len(raw) == 0 {
			return false
		}
		var r struct {
			Status string `json:"status"`
		}
		return json.Unmarshal(raw, &r) == nil && r.Status == "0x0"
	}, 60*time.Second, time.Second, "claim with zero refundable reverts (status 0x0)")
}
