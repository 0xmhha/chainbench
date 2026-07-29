// This file ports the go-stablenet GovCouncil access-control proposal lifecycles
// (from regression f-system-contracts f5-01 blacklist and f5-03
// authorize), driven through node-side signing (each validator votes from its
// own unlocked coinbase).
//
// # Test: blacklist-proposal-executes
//
// Intent:   a GovCouncil proposeAddBlacklist proposal, approved by validators to
//
//	quorum, executes: the target becomes blacklisted and an AddressBlacklisted
//	event is emitted.
//
// Applies:  stablenet. Requires "rpc".
// Method:   propose proposeAddBlacklist(target) from the first validator on
//
//	GovCouncil; extract the proposalId; approve from the other validators to
//	quorum; read AccountManager.isBlacklisted(target) and check the
//	AddressBlacklisted(address,uint256) log for the target.
//
// Pass:     isBlacklisted(target) == 1 and an AddressBlacklisted event is emitted.
//
// # Test: authorize-proposal-executes
//
// Intent:   a GovCouncil proposeAddAuthorizedAccount proposal, approved to quorum,
//
//	executes and the target becomes authorized.
//
// Applies:  stablenet. Requires "rpc".
// Method:   propose proposeAddAuthorizedAccount(target) from the first validator;
//
//	extract the proposalId; approve to quorum; read
//	AccountManager.isAuthorized(target).
//
// Pass:     isAuthorized(target) == 1 afterward.
//
// These are chainbench TEST CODE (requirement #16): live multi-validator
// governance flows (the sibling _test.go validates registration/gating). They
// need at least two validators for quorum.
package anzeon

import (
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// govCouncil is the go-stablenet GovCouncil system contract (regression
// lib/common.sh GOV_COUNCIL).
const govCouncil = "0x0000000000000000000000000000000000001004"

// blacklistTarget and authorizeTarget are fixed fixture addresses acted on by
// the two proposals; the AccountManager mappings accept any address.
const (
	blacklistTarget = "0x00000000000000000000000000000000C0FFEE06"
	authorizeTarget = "0x00000000000000000000000000000000C0FFEE0A"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "blacklist-proposal-executes",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           blacklistProposalExecutes,
	})
	testkit.Register(testkit.Case{
		Name:         "authorize-proposal-executes",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           authorizeProposalExecutes,
	})
}

func blacklistProposalExecutes(t *testkit.T) {
	proposer := councilProposalToQuorum(t, "proposeAddBlacklist(address)", blacklistTarget)

	// The AccountManager now reports the target as blacklisted.
	t.WaitFor(func() bool {
		v, err := accounts.ReadUint(t.Ctx(), proposer.client.EthCall, accountManager,
			"isBlacklisted(address)", accounts.AddressArg(blacklistTarget))
		return err == nil && v.Cmp(big.NewInt(1)) == 0
	}, 60*time.Second, time.Second, "target is blacklisted")

	// GovCouncil emits AddressBlacklisted(address indexed, uint256 indexed) for it.
	t.Truef(emittedEventForTarget(t, proposer.client, govCouncil,
		"AddressBlacklisted(address,uint256)", blacklistTarget),
		"AddressBlacklisted event emitted for the target")
}

func authorizeProposalExecutes(t *testkit.T) {
	proposer := councilProposalToQuorum(t, "proposeAddAuthorizedAccount(address)", authorizeTarget)

	// The AccountManager now reports the target as authorized.
	t.WaitFor(func() bool {
		v, err := accounts.ReadUint(t.Ctx(), proposer.client.EthCall, accountManager,
			"isAuthorized(address)", accounts.AddressArg(authorizeTarget))
		return err == nil && v.Cmp(big.NewInt(1)) == 0
	}, 60*time.Second, time.Second, "target is authorized")
}

// councilProposalToQuorum proposes proposeSig(target) on GovCouncil from the
// first validator and approves it to quorum, returning the proposer for the
// post-execution reads.
func councilProposalToQuorum(t *testkit.T, proposeSig, target string) validator {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]

	data := accounts.EncodeCallArgs(proposeSig, accounts.Address(target))
	proposeHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govCouncil, Data: data, Gas: govGas,
	})
	t.NoErr(err, proposeSig)

	proposalID := extractProposalID(t, proposer, proposeHash)
	approveToQuorum(t, govCouncil, proposalID, proposer, vals[1:])
	return proposer
}

// emittedEventForTarget reports whether a log with topic0 == keccak(eventSig) and
// the target address in its indexed topics was emitted by contract. It filters by
// topic0 and the left-padded target topic via eth_getLogs.
func emittedEventForTarget(t *testkit.T, c *rpc.Client, contract, eventSig, target string) bool {
	var logs []accounts.Log
	err := c.Call(t.Ctx(), "eth_getLogs", &logs, map[string]any{
		"address":   contract,
		"fromBlock": "earliest",
		"toBlock":   "latest",
		"topics":    []any{accounts.EventTopic(eventSig), addrTopic(target)},
	})
	return err == nil && len(logs) > 0
}

// addrTopic left-pads a 0x-hex address to a 32-byte (64-hex) indexed-topic word.
func addrTopic(addr string) string {
	a := strings.ToLower(strings.TrimPrefix(addr, "0x"))
	if len(a) >= 64 {
		return "0x" + a
	}
	return "0x" + strings.Repeat("0", 64-len(a)) + a
}
