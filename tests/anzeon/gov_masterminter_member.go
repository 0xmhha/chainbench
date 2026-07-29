// This file ports the GovMasterMinter self-member add/remove case (from
// regression f-system-contracts f4-03). GovMasterMinter can add and remove
// its own governance members, changing the quorum. isActiveMember is a modifier
// (not callable), so membership is read via isMember(address, memberVersion).
//
// # Test: masterminter-member-add-remove (f4-03)
//
// Intent:   proposeAddMember raises the quorum and adds a member; proposeRemoveMember
//
//	restores the quorum and removes it.
//
// Applies:  stablenet. Requires "rpc".
// Method:   add a FRESH member with newQuorum=3 to quorum; assert isMember==true
//
//	and quorum==3; remove it with newQuorum=2; assert isMember==false and
//	quorum==2. The fresh member and the restore keep the case self-contained.
//
// This is chainbench TEST CODE (requirement #16): a live governance flow that
// perturbs GovMasterMinter membership/quorum, so the sibling _test.go validates
// registration/gating.
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "masterminter-member-add-remove",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           masterMinterMemberAddRemove,
	})
}

func masterMinterMemberAddRemove(t *testkit.T) {
	vals := discoverValidators(t)
	proposer := vals[0]

	_, member, err := accounts.GenerateKey()
	t.NoErr(err, "generate member")

	// Add the member (newQuorum is uint32), raising the quorum to 3.
	masterMinterProposeToQuorum(t, proposer, vals, "proposeAddMember(address,uint32)",
		accounts.Address(member), accounts.Uint(big.NewInt(3)))
	t.WaitFor(func() bool { return masterMinterIsMember(t, member) && masterMinterQuorum(t) == 3 },
		90*time.Second, time.Second, "member added and quorum == 3")

	// Remove it, restoring the quorum to 2.
	masterMinterProposeToQuorum(t, proposer, vals, "proposeRemoveMember(address,uint32)",
		accounts.Address(member), accounts.Uint(big.NewInt(2)))
	t.WaitFor(func() bool { return !masterMinterIsMember(t, member) && masterMinterQuorum(t) == 2 },
		90*time.Second, time.Second, "member removed and quorum == 2")
}

// masterMinterIsMember reads isMember(member, memberVersion) on GovMasterMinter,
// re-reading memberVersion each call since it bumps as members change.
func masterMinterIsMember(t *testkit.T, member string) bool {
	ver, err := accounts.ReadUint(t.Ctx(), caller(t), govMasterMinter, "memberVersion()")
	if err != nil {
		return false
	}
	v, err := accounts.ReadUint(t.Ctx(), caller(t), govMasterMinter, "isMember(address,uint256)",
		accounts.AddressArg(member), uint256Arg(ver))
	return err == nil && v.Cmp(big.NewInt(1)) == 0
}

// masterMinterQuorum reads GovMasterMinter.quorum(), or -1 on error.
func masterMinterQuorum(t *testkit.T) int64 {
	v, err := accounts.ReadUint(t.Ctx(), caller(t), govMasterMinter, "quorum()")
	if err != nil {
		return -1
	}
	return v.Int64()
}

// uint256Arg encodes n as a 32-byte big-endian ABI word for ReadUint.
func uint256Arg(n *big.Int) []byte {
	b := make([]byte, 32)
	n.FillBytes(b)
	return b
}
