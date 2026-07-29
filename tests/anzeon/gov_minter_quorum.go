// This file ports the remove-minter and quorum-deficient governance cases (from
// regression f-system-contracts f4-02, f2-03).
//
// # Test: remove-minter-executes (f4-02)
//
// Intent:   GovMasterMinter can remove a configured minter.
// Applies:  stablenet. Requires "rpc".
// Method:   configure a FRESH minter (proposeConfigureMinter to quorum), confirm
//
//	NativeCoinAdapter.isMinter==1, then proposeRemoveMinter to quorum and confirm
//	isMinter==0. The fresh minter keeps the case self-contained.
//
// # Test: quorum-deficient-stays-voting (f2-03)
//
// Intent:   a proposal executed before reaching quorum reverts and stays Voting.
// Applies:  stablenet. Requires "rpc".
// Method:   proposeMint (the proposer auto-casts one vote, below the quorum of 2),
//
//	call executeProposal immediately — it reverts — and read the proposal status
//	== Voting(1). Clean up by approving to quorum so no Voting proposal dangles.
//
// These are chainbench TEST CODE (requirement #16): live governance flows, so the
// sibling _test.go validates registration/gating.
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	reg := func(name string, fn func(*testkit.T)) {
		testkit.Register(testkit.Case{
			Name:         name,
			Category:     "system-contracts",
			ChainCompat:  []string{"stablenet"},
			RequiresCaps: []string{"rpc"},
			Fn:           fn,
		})
	}
	reg("remove-minter-executes", removeMinterExecutes)
	reg("quorum-deficient-stays-voting", quorumDeficientStaysVoting)
}

// masterMinterProposeToQuorum runs a GovMasterMinter proposal (sig + args) to
// quorum via node-side signing.
func masterMinterProposeToQuorum(t *testkit.T, proposer validator, vals []validator, sig string, args ...accounts.Arg) {
	data := accounts.EncodeCallArgs(sig, args...)
	hash, err := proposer.client.SendTransaction(t.Ctx(), rpc.SendTxArgs{
		From: proposer.addr, To: govMasterMinter, Data: data, Gas: govGas,
	})
	t.NoErr(err, sig)
	id := extractProposalID(t, proposer, hash)
	approveToQuorum(t, govMasterMinter, id, proposer, vals[1:])
}

// isMinterReaches waits until NativeCoinAdapter.isMinter(minter) == want.
func isMinterReaches(t *testkit.T, minter string, want int64) {
	t.WaitFor(func() bool {
		v, err := accounts.ReadUint(t.Ctx(), caller(t), nativeCoinAdapter,
			"isMinter(address)", accounts.AddressArg(minter))
		return err == nil && v.Cmp(big.NewInt(want)) == 0
	}, 60*time.Second, time.Second, "isMinter reaches the expected value")
}

func removeMinterExecutes(t *testkit.T) {
	vals := discoverValidators(t)
	proposer := vals[0]

	_, minter, err := accounts.GenerateKey()
	t.NoErr(err, "generate minter")

	// Configure a fresh minter, then remove it.
	allowance := new(big.Int).Mul(big.NewInt(10), big.NewInt(1_000_000_000_000_000_000)) // 10 coins
	masterMinterProposeToQuorum(t, proposer, vals, "proposeConfigureMinter(address,uint256)",
		accounts.Address(minter), accounts.Uint(allowance))
	isMinterReaches(t, minter, 1)

	masterMinterProposeToQuorum(t, proposer, vals, "proposeRemoveMinter(address)", accounts.Address(minter))
	isMinterReaches(t, minter, 0)
}

func quorumDeficientStaysVoting(t *testkit.T) {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]

	_, beneficiary, err := accounts.GenerateKey()
	t.NoErr(err, "generate beneficiary")

	amount := big.NewInt(1_000_000_000_000_000_000)
	proof := govbind.MintProof(beneficiary, amount, big.NewInt(time.Now().Unix()),
		"REG-DEP", "REG-BANK", "quorum-deficient test")
	proposeHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.ProposeMintCall(proof), Gas: govGas,
	})
	t.NoErr(err, "proposeMint")
	id := extractProposalID(t, proposer, proposeHash)

	// Execute before quorum (only the proposer's auto-vote) — it must revert.
	exHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.ExecuteProposalCall(id), Gas: govGas,
	})
	if err == nil {
		t.WaitFor(func() bool { return receiptReverted(ctx, proposer.client, exHash) },
			60*time.Second, time.Second, "premature execute reverts (status 0x0)")
	}

	// The proposal is still Voting (not executed).
	ret, err := proposer.client.EthCall(ctx, govMinter, govbind.ProposalsCall(id))
	t.NoErr(err, "proposals")
	st, ok := govbind.DecodeProposalStatus(ret)
	t.Truef(ok && st == govbind.ProposalVoting,
		"proposal stays Voting after a quorum-deficient execute (status=%d ok=%v)", st, ok)

	// Clean up: approve to quorum so no Voting proposal dangles for later cases.
	approveToQuorum(t, govMinter, id, proposer, vals[1:])
}
