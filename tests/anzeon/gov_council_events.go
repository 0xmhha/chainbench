// This file ports the GovCouncil authorize/blacklist lifecycle event cases (from
// tests/regression f-system-contracts f5-04, f5-07, f5-08, f5-09). Each grants a
// status to a FRESH generated account via governance, then reverses it, asserting
// both the AccountManager state and the GovCouncil lifecycle event. Using a fresh
// target keeps each case independent of the others in a full run.
//
// # Test: authorized-account-added-event (f5-08)
//
// Intent:   authorizing an account emits AuthorizedAccountAdded.
// Applies:  stablenet. Requires "rpc".
// Method:   proposeAddAuthorizedAccount to quorum; assert isAuthorized==1 and an
//
//	AuthorizedAccountAdded(address,uint256) log for the target.
//
// # Test: unauthorize-proposal-executes (f5-04 + f5-09)
//
// Intent:   removing an authorized account clears the status and emits
//
//	AuthorizedAccountRemoved.
//
// Applies:  stablenet. Requires "rpc".
// Method:   authorize a fresh account, then proposeRemoveAuthorizedAccount to
//
//	quorum; assert isAuthorized==0 and an AuthorizedAccountRemoved log.
//
// # Test: address-unblacklisted-event (f5-07)
//
// Intent:   removing a blacklist emits AddressUnblacklisted.
// Applies:  stablenet. Requires "rpc".
// Method:   blacklist a fresh account, then proposeRemoveBlacklist to quorum;
//
//	assert isBlacklisted==0 and an AddressUnblacklisted log.
//
// These are chainbench TEST CODE (requirement #16): live governance flows, so the
// sibling _test.go validates registration/gating.
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
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
	reg("authorized-account-added-event", authorizedAccountAddedEvent)
	reg("unauthorize-proposal-executes", unauthorizeProposalExecutes)
	reg("address-unblacklisted-event", addressUnblacklistedEvent)
}

// amStatusReaches waits until AccountManager.<method>(addr) equals want.
func amStatusReaches(t *testkit.T, method, addr string, want int64) {
	t.WaitFor(func() bool {
		v, err := accounts.ReadUint(t.Ctx(), caller(t), accountManager, method, accounts.AddressArg(addr))
		return err == nil && v.Cmp(big.NewInt(want)) == 0
	}, 60*time.Second, time.Second, method+" reaches the expected value")
}

func authorizedAccountAddedEvent(t *testkit.T) {
	_, target, err := accounts.GenerateKey()
	t.NoErr(err, "generate target")

	proposer := councilProposalToQuorum(t, "proposeAddAuthorizedAccount(address)", target)
	amStatusReaches(t, "isAuthorized(address)", target, 1)
	t.Truef(emittedEventForTarget(t, proposer.client, govCouncil,
		"AuthorizedAccountAdded(address,uint256)", target),
		"AuthorizedAccountAdded event emitted for the target")
}

func unauthorizeProposalExecutes(t *testkit.T) {
	_, target, err := accounts.GenerateKey()
	t.NoErr(err, "generate target")

	councilProposalToQuorum(t, "proposeAddAuthorizedAccount(address)", target)
	amStatusReaches(t, "isAuthorized(address)", target, 1)

	proposer := councilProposalToQuorum(t, "proposeRemoveAuthorizedAccount(address)", target)
	amStatusReaches(t, "isAuthorized(address)", target, 0)
	t.Truef(emittedEventForTarget(t, proposer.client, govCouncil,
		"AuthorizedAccountRemoved(address,uint256)", target),
		"AuthorizedAccountRemoved event emitted for the target")
}

func addressUnblacklistedEvent(t *testkit.T) {
	_, target, err := accounts.GenerateKey()
	t.NoErr(err, "generate target")

	councilProposalToQuorum(t, "proposeAddBlacklist(address)", target)
	amStatusReaches(t, "isBlacklisted(address)", target, 1)

	proposer := councilProposalToQuorum(t, "proposeRemoveBlacklist(address)", target)
	amStatusReaches(t, "isBlacklisted(address)", target, 0)
	t.Truef(emittedEventForTarget(t, proposer.client, govCouncil,
		"AddressUnblacklisted(address,uint256)", target),
		"AddressUnblacklisted event emitted for the target")
}
