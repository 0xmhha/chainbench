// This file ports the NativeCoinAdapter mint/burn Transfer-event cases (from
// regression f-system-contracts f1-04, f1-05). A governance mint emits
// Transfer(0x0 -> beneficiary) and a burn emits Transfer(account -> 0x0) on the
// NativeCoinAdapter. Each runs the governance flow and asserts the event via
// eth_getLogs.
//
// # Test: mint-transfer-event (f1-04)
//
// Intent:   a mint emits Transfer(0x0 -> beneficiary).
// Applies:  stablenet. Requires "rpc".
// Method:   proposeMint to a FRESH beneficiary, approve to quorum; assert a
//
//	Transfer log on the NativeCoinAdapter with from=0x0 and to=beneficiary.
//
// # Test: burn-transfer-event (f1-05)
//
// Intent:   a burn emits Transfer(account -> 0x0).
// Applies:  stablenet. Requires "rpc".
// Method:   count Transfer-to-0x0 logs, run a burn to execution, assert the count
//
//	increased (the burn account is set by the proof, so match to=0x0 only).
//
// These are chainbench TEST CODE (requirement #16): live governance flows, so the
// sibling _test.go validates registration/gating.
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// zeroAddr is the mint source / burn destination in a Transfer event.
const zeroAddr = "0x0000000000000000000000000000000000000000"

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
	reg("mint-transfer-event", mintTransferEvent)
	reg("burn-transfer-event", burnTransferEvent)
}

// transferLogCount returns how many NativeCoinAdapter Transfer logs match the
// from/to filter ("" = any for that position), or -1 on RPC error.
func transferLogCount(t *testkit.T, c *rpc.Client, from, to string) int {
	topics := []any{accounts.EventTopic("Transfer(address,address,uint256)"), topicOrNil(from), topicOrNil(to)}
	var logs []accounts.Log
	if err := c.Call(t.Ctx(), "eth_getLogs", &logs, map[string]any{
		"address":   nativeCoinAdapter,
		"fromBlock": "earliest",
		"toBlock":   "latest",
		"topics":    topics,
	}); err != nil {
		return -1
	}
	return len(logs)
}

// topicOrNil maps an empty address to a null topic (matches any).
func topicOrNil(addr string) any {
	if addr == "" {
		return nil
	}
	return addrTopic(addr)
}

func mintTransferEvent(t *testkit.T) {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]

	_, beneficiary, err := accounts.GenerateKey()
	t.NoErr(err, "generate beneficiary")

	amount := big.NewInt(1_000_000_000_000_000_000)
	proof := govbind.MintProof(beneficiary, amount, big.NewInt(time.Now().Unix()),
		"REG-DEP", "REG-BANK", "regression mint event")
	proposeHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.ProposeMintCall(proof), Gas: govGas,
	})
	t.NoErr(err, "proposeMint")

	proposalID := extractProposalID(t, proposer, proposeHash)
	approveToQuorum(t, govMinter, proposalID, proposer, vals[1:])

	// A fresh beneficiary makes this Transfer(0x0 -> beneficiary) unique to this mint.
	t.WaitFor(func() bool { return transferLogCount(t, proposer.client, zeroAddr, beneficiary) > 0 },
		60*time.Second, time.Second, "mint Transfer(0x0 -> beneficiary) event")
}

func burnTransferEvent(t *testkit.T) {
	vals := discoverValidators(t)
	proposer := vals[0]

	before := transferLogCount(t, proposer.client, "", zeroAddr)
	t.Truef(before >= 0, "read Transfer-to-0x0 count before burn")

	proposalID := proposeBurnFrom(t, proposer)
	approveToQuorum(t, govMinter, proposalID, proposer, vals[1:])

	// The burn account is fixed by the proof, so match to=0x0 only and assert the
	// count increased — a new Transfer(account -> 0x0) was emitted by this burn.
	t.WaitFor(func() bool { return transferLogCount(t, proposer.client, "", zeroAddr) > before },
		60*time.Second, time.Second, "burn Transfer(account -> 0x0) event count increases")
}
