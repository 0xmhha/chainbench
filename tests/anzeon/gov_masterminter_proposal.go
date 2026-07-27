// This file ports the go-stablenet GovMasterMinter configure-minter proposal
// lifecycle (from tests/regression f-system-contracts f4-01), driven through
// node-side signing (each validator votes from its own unlocked coinbase).
//
// # Test: configure-minter-proposal-executes
//
// Intent:   a GovMasterMinter proposeConfigureMinter(minter, allowance) proposal,
//
//	approved by validators to quorum, executes and grants the minter a
//	non-zero allowance on the native-coin adapter.
//
// Applies:  stablenet. Requires "rpc".
// Method:   propose proposeConfigureMinter(minter, allowance) from the first
//
//	validator on GovMasterMinter; extract the proposalId; approve from the
//	other validators to quorum; read the adapter's minterAllowance(minter).
//
// Pass:     minterAllowance(minter) > 0 afterward.
//
// This is chainbench TEST CODE (requirement #16): a live multi-validator
// governance flow (the sibling _test.go validates registration/gating). It needs
// at least two validators for quorum.
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// govMasterMinter is the go-stablenet GovMasterMinter system contract (regression
// lib/common.sh GOV_MASTER_MINTER).
const govMasterMinter = "0x0000000000000000000000000000000000001002"

// minterAccount is the fixture address configured as a minter by the proposal.
const minterAccount = "0x00000000000000000000000000000000C0FFEE07"

func init() {
	testkit.Register(testkit.Case{
		Name:         "configure-minter-proposal-executes",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           configureMinterProposalExecutes,
	})
}

func configureMinterProposalExecutes(t *testkit.T) {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]

	allowance := new(big.Int).Mul(big.NewInt(10), big.NewInt(1_000_000_000_000_000_000)) // 10 coins
	data := accounts.EncodeCallArgs("proposeConfigureMinter(address,uint256)",
		accounts.Address(minterAccount), accounts.Uint(allowance))
	proposeHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govMasterMinter, Data: data, Gas: govGas,
	})
	t.NoErr(err, "proposeConfigureMinter")

	proposalID := extractProposalID(t, proposer, proposeHash)
	approveToQuorum(t, govMasterMinter, proposalID, proposer, vals[1:])

	// The adapter now grants the minter a non-zero allowance.
	t.WaitFor(func() bool {
		a, err := accounts.ReadUint(ctx, proposer.client.EthCall, nativeCoinAdapter,
			"minterAllowance(address)", accounts.AddressArg(minterAccount))
		return err == nil && a.Sign() > 0
	}, 60*time.Second, time.Second, "minter has a non-zero minterAllowance")
}
