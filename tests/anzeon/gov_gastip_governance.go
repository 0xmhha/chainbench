// This file ports the GasTip governance case (from regression b-wbft b-06
// and f-system-contracts f3-05). Changing the GasTip through GovValidator
// governance updates the block header's WBFTExtra.GasTip and emits GasTipUpdated;
// the case then restores the original value. The gastip-read cases (c-01/c-02)
// read the header GasTip dynamically, so this perturbation does not break them.
//
// # Test: gastip-governance-updates-header (b-06 + f3-05)
//
// Intent:   a GovValidator proposeGasTip lifecycle changes the header GasTip and
//
//	emits GasTipUpdated, and the value can be restored.
//
// Applies:  stablenet. Requires "rpc".
// Method:   read the current header GasTip; propose a DIFFERENT value to quorum
//
//	(avoiding SameGasTip); assert the header reflects it and a GasTipUpdated log
//	was emitted; then propose the original value back and assert it is restored.
//
// This is chainbench TEST CODE (requirement #16): a live governance flow that
// perturbs a network parameter, so the sibling _test.go validates
// registration/gating.
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// Known WBFTExtra GasTip values (wei) from the regression profile.
var (
	gastipDefault = big.NewInt(27600000000000)
	gastipNew     = big.NewInt(30000000000000)
)

const gasTipUpdatedSig = "GasTipUpdated(uint256,uint256,address)"

func init() {
	testkit.Register(testkit.Case{
		Name:         "gastip-governance-updates-header",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           gastipGovernanceUpdatesHeader,
	})
}

func gastipGovernanceUpdatesHeader(t *testkit.T) {
	vals := discoverValidators(t)
	proposer := vals[0]

	cur := headerGasTip(t, latestBlockHex(t))
	t.Truef(cur.Sign() > 0, "current header GasTip is positive (got %s)", cur)

	// Pick a target different from the current value to avoid a SameGasTip revert.
	target := gastipNew
	if cur.Cmp(gastipNew) == 0 {
		target = gastipDefault
	}

	eventsBefore := logCountByTopic0(t, proposer.client, govValidator, gasTipUpdatedSig)

	proposeGasTipToQuorum(t, proposer, vals, target)
	waitHeaderGasTip(t, target)

	// GasTipUpdated was emitted by GovValidator (f3-05).
	t.WaitFor(func() bool {
		return logCountByTopic0(t, proposer.client, govValidator, gasTipUpdatedSig) > eventsBefore
	}, 30*time.Second, time.Second, "GasTipUpdated event emitted")

	// Restore the original value (b-06 revert).
	proposeGasTipToQuorum(t, proposer, vals, cur)
	waitHeaderGasTip(t, cur)
}

// proposeGasTipToQuorum runs a GovValidator proposeGasTip(tip) proposal to quorum.
func proposeGasTipToQuorum(t *testkit.T, proposer validator, vals []validator, tip *big.Int) {
	data := accounts.EncodeCallArgs("proposeGasTip(uint256)", accounts.Uint(tip))
	hash, err := proposer.client.SendTransaction(t.Ctx(), rpc.SendTxArgs{
		From: proposer.addr, To: govValidator, Data: data, Gas: govGas,
	})
	t.NoErr(err, "proposeGasTip")
	id := extractProposalID(t, proposer, hash)
	approveToQuorum(t, govValidator, id, proposer, vals[1:])
}

// waitHeaderGasTip waits until the latest block's WBFTExtra.GasTip equals want.
func waitHeaderGasTip(t *testkit.T, want *big.Int) {
	t.WaitFor(func() bool { return headerGasTip(t, latestBlockHex(t)).Cmp(want) == 0 },
		60*time.Second, time.Second, "header GasTip reaches the proposed value")
}

// latestBlockHex returns the latest block number as a 0x-hex string.
func latestBlockHex(t *testkit.T) string {
	var latest string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_blockNumber", &latest), "eth_blockNumber")
	return latest
}

// logCountByTopic0 counts a contract's logs with the given event's topic0.
func logCountByTopic0(t *testkit.T, c *rpc.Client, contract, eventSig string) int {
	var logs []accounts.Log
	if err := c.Call(t.Ctx(), "eth_getLogs", &logs, map[string]any{
		"address":   contract,
		"fromBlock": "earliest",
		"toBlock":   "latest",
		"topics":    []any{accounts.EventTopic(eventSig)},
	}); err != nil {
		return -1
	}
	return len(logs)
}
