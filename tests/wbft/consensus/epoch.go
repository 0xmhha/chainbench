// This file ports the epoch-transition case (from regression b-wbft b-03).
// At an epoch boundary block the WBFT extra data carries an epochInfo snapshot
// (the validator/candidate set for the new epoch); non-boundary blocks do not.
//
// # Test: epoch-transition-carries-epoch-info
//
// Intent:   the block at an epoch boundary carries an epochInfo with a non-empty
//
//	validator/candidate set.
//
// Applies:  stablenet (epochLength 140). Requires "rpc".
// Method:   compute the next epoch boundary block, wait for the chain to reach
//
//	it, read istanbul_getWbftExtraInfo at that block, and assert epochInfo is
//	present and lists validators/candidates.
//
// This is chainbench TEST CODE (requirement #16): it observes a live chain across
// an epoch boundary (a slow, read-only case), so the sibling _test.go validates
// registration/gating.
package consensus

import (
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/testkit"
)

// epochLength is the stablenet regression epoch length (genesis anzeon.wbft).
const epochLength = 140

func init() {
	testkit.Register(testkit.Case{
		Name:         "epoch-transition-carries-epoch-info",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           epochTransitionCarriesEpochInfo,
	})
}

func epochTransitionCarriesEpochInfo(t *testkit.T) {
	cur, err := t.Primary().BlockNumber(t.Ctx())
	t.NoErr(err, "eth_blockNumber")

	// Next epoch boundary strictly after the current block.
	epochBlock := (cur/epochLength + 1) * epochLength
	t.WaitFor(func() bool {
		n, e := t.Primary().BlockNumber(t.Ctx())
		return e == nil && n >= epochBlock
	}, 220*time.Second, 2*time.Second, "chain reaches the epoch boundary block")

	var extra struct {
		EpochInfo *struct {
			Validators []string `json:"validators"`
			Candidates []string `json:"candidates"`
		} `json:"epochInfo"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getWbftExtraInfo", &extra, fmt.Sprintf("0x%x", epochBlock)),
		"istanbul_getWbftExtraInfo(epoch)")

	t.Truef(extra.EpochInfo != nil, "epoch boundary block %d carries an epochInfo snapshot", epochBlock)
	if extra.EpochInfo != nil {
		set := len(extra.EpochInfo.Validators) + len(extra.EpochInfo.Candidates)
		t.Truef(set > 0, "epochInfo lists a non-empty validator/candidate set (got %d)", set)
	}
}
