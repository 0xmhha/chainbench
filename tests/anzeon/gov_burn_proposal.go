// This file ports the go-stablenet governance burn-proposal lifecycle (from
// regression f-system-contracts f2-02), a payable proposeBurn on
// GovMinter via node-side signing.
//
// # Test: burn-proposal-executes
//
// Intent:   a GovMinter burn proposal (payable; the proposer is the burn target
//
//	and sends msg.value == amount), approved to quorum, executes.
//
// Applies:  stablenet. Requires "rpc".
// Method:   propose proposeBurn(proof) from the first validator with value ==
//
//	amount and from == that validator; extract the proposalId; approve from
//	the other validators to quorum; assert the proposal reaches Executed.
//
// Pass:     the proposal's proposals() status is Executed.
//
// This is chainbench TEST CODE (requirement #16): a live multi-validator
// governance flow (the sibling _test.go validates registration/gating).
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/internal/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "burn-proposal-executes",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           burnProposalExecutes,
	})
}

func burnProposalExecutes(t *testkit.T) {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]

	// A modest amount so the proposer's coinbase (genesis-funded) can cover the
	// payable value. The proposer must be the burn target (BurnFromMustBeProposer).
	amount := big.NewInt(1_000_000_000_000_000) // 0.001 coin
	proof := govbind.BurnProof(proposer.addr, amount, big.NewInt(time.Now().Unix()),
		"REG-WD", "REG-REF", "regression burn")
	proposeHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.ProposeBurnCall(proof),
		Gas: govGas, Value: "0x" + amount.Text(16),
	})
	t.NoErr(err, "proposeBurn")

	proposalID := extractProposalID(t, proposer, proposeHash)
	approveToQuorum(t, govMinter, proposalID, proposer, vals[1:])

	// The proposal executed.
	t.WaitFor(func() bool { return proposalExecuted(ctx, proposer.client, govMinter, proposalID) },
		60*time.Second, time.Second, "burn proposal reaches Executed")
}
