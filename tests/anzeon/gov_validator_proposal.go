// This file ports the go-stablenet validator add-member governance lifecycle
// (from regression f-system-contracts f3-01/f3-02), on the GovValidator
// system contract via node-side signing.
//
// # Test: validator-add-member-executes
//
// Intent:   a GovValidator proposeAddMember proposal, approved by validators to
//
//	quorum, executes and the new address becomes an active member.
//
// Applies:  stablenet. Requires "rpc".
// Method:   propose proposeAddMember(newMember, newQuorum) from the first
//
//	validator on GovValidator; extract the proposalId; approve from the other
//	validators to quorum; read members(newMember) and assert it is active.
//
// Pass:     members(newMember) reports the member as active (1).
//
// This is chainbench TEST CODE (requirement #16): a live multi-validator
// governance flow (the sibling _test.go validates registration/gating).
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// newValidatorMember is the preset node5 (endpoint) address — a real fixture
// account not already a GovValidator member (regression acct 5).
const newValidatorMember = "0x5400d8b543eaf6738c7b44799623bea88fd0f5ee"

func init() {
	testkit.Register(testkit.Case{
		Name:         "validator-add-member-executes",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           validatorAddMemberExecutes,
	})
}

func validatorAddMemberExecutes(t *testkit.T) {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]

	// proposeAddMember(newMember, newQuorum). Keep the quorum at the current
	// validator count so the set stays operable after the add.
	data := govbind.ProposeAddMemberCall(newValidatorMember, uint32(len(vals)))
	proposeHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govValidator, Data: data, Gas: govGas,
	})
	t.NoErr(err, "proposeAddMember")

	proposalID := extractProposalID(t, proposer, proposeHash)
	approveToQuorum(t, govValidator, proposalID, proposer, vals[1:])

	// The new address is now an active member: members(address) returns a tuple
	// whose first word (isActive) is 1.
	t.WaitFor(func() bool {
		active, err := accounts.ReadUint(ctx, proposer.client.EthCall, govValidator,
			"members(address)", accounts.AddressArg(newValidatorMember))
		return err == nil && active.Cmp(big.NewInt(1)) == 0
	}, 60*time.Second, time.Second, "new address is an active GovValidator member")
}
