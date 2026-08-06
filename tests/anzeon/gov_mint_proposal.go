// This file ports the go-stablenet governance mint-proposal lifecycle (from
// regression f-system-contracts f2-01), exercising node-side signing (a
// validator's unlocked coinbase votes) plus the govbind calldata/decoders.
//
// # Test: mint-proposal-executes
//
// Intent:   a GovMinter mint proposal, proposed and approved by validators to
//
//	quorum, executes and credits the beneficiary with the minted amount.
//
// Applies:  stablenet (the go-stablenet GovBase system contracts). Requires "rpc".
// Method:   discover each validator's coinbase (eth_coinbase); propose
//
//	proposeMint(proof) from the first validator; extract the proposalId from
//	the ProposalCreated log; approve from the other validators to quorum
//	(auto-executing, else execute manually).
//
// Pass:     the beneficiary's balance increases by exactly the minted amount.
//
// This is chainbench TEST CODE (requirement #16): it drives real governance
// transactions signed node-side by validators, so it is only meaningful against
// a live multi-validator network (the sibling _test.go validates
// registration/gating). It needs at least two validators for quorum.
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/internal/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// mintBeneficiary starts unfunded so the minted amount is unambiguous.
const mintBeneficiary = "0x00000000000000000000000000000000C0FFEE05"

func init() {
	testkit.Register(testkit.Case{
		Name:         "mint-proposal-executes",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           mintProposalExecutes,
	})
}

func mintProposalExecutes(t *testkit.T) {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]

	balBefore, err := proposer.client.BalanceAt(ctx, mintBeneficiary)
	t.NoErr(err, "balance before")

	amount := big.NewInt(1_000_000_000_000_000_000) // 1 coin
	proof := govbind.MintProof(mintBeneficiary, amount, big.NewInt(time.Now().Unix()),
		"REG-DEP", "REG-BANK", "regression mint")
	proposeHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.ProposeMintCall(proof), Gas: govGas,
	})
	t.NoErr(err, "proposeMint")

	proposalID := extractProposalID(t, proposer, proposeHash)
	approveToQuorum(t, govMinter, proposalID, proposer, vals[1:])

	// The beneficiary is credited the minted amount.
	t.WaitFor(func() bool {
		balAfter, err := proposer.client.BalanceAt(ctx, mintBeneficiary)
		return err == nil && new(big.Int).Sub(balAfter, balBefore).Cmp(amount) == 0
	}, 60*time.Second, time.Second, "beneficiary credited the minted amount")
}
