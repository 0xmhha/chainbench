package poa_test

import (
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
)

// The governance policy in DefaultEnv is not a choice this code made. It is the
// policy a live wemix->wbft handoff was verified with, restated in Go so that a
// network can be composed without an operator supplying thirteen parameters.
//
// testdata/verified-governance.json is that record. It used to live in a
// hardfork preset the DSL loaded, which made a document a test author reads
// carry a value only this package consumes; the document is gone and the record
// moved to the thing it guards.

// TestDefaultEnv_IsStillWhatWasVerified fails when the code's defaults drift
// from the recorded run.
//
// A failure here is not a mistake to undo. It means this package has started
// CHOOSING a policy rather than restating one, and that is precisely the point
// at which a governance policy needs a way to be DECLARED — a surface for it,
// which is deliberately not built while nothing asks for one.
func TestDefaultEnv_IsStillWhatWasVerified(t *testing.T) {
	raw, err := os.ReadFile("testdata/verified-governance.json")
	if err != nil {
		t.Fatalf("read the verified record: %v", err)
	}
	var rec struct {
		BallotDurationMin    int64   `json:"ballot_duration_min"`
		BallotDurationMax    int64   `json:"ballot_duration_max"`
		StakingMin           string  `json:"staking_min"`
		StakingMax           string  `json:"staking_max"`
		MaxIdleBlockInterval int64   `json:"max_idle_block_interval"`
		BlockCreationTime    int64   `json:"block_creation_time"`
		BlockRewardAmount    string  `json:"block_reward_amount"`
		MaxPriorityFeePerGas string  `json:"max_priority_fee_per_gas"`
		RewardDistribution   []int   `json:"reward_distribution"`
		MaxBaseFee           string  `json:"max_base_fee"`
		BlockGasLimit        int64   `json:"block_gas_limit"`
		BaseFeeMaxChangeRate int64   `json:"base_fee_max_change_rate"`
		GasTargetPercentage  int64   `json:"gas_target_percentage"`
		Unknown              *string `json:"-"`
	}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("parse the verified record: %v", err)
	}
	want := poa.Env{
		BallotDurationMin: rec.BallotDurationMin, BallotDurationMax: rec.BallotDurationMax,
		StakingMin: dec(t, rec.StakingMin), StakingMax: dec(t, rec.StakingMax),
		MaxIdleBlockInterval: rec.MaxIdleBlockInterval, BlockCreationTime: rec.BlockCreationTime,
		BlockRewardAmount: dec(t, rec.BlockRewardAmount), MaxPriorityFeePerGas: dec(t, rec.MaxPriorityFeePerGas),
		RewardDistribution: rec.RewardDistribution, MaxBaseFee: dec(t, rec.MaxBaseFee),
		BlockGasLimit: rec.BlockGasLimit, BaseFeeMaxChangeRate: rec.BaseFeeMaxChangeRate,
		GasTargetPercentage: rec.GasTargetPercentage,
	}
	if diff := envDiff(want, poa.DefaultEnv()); diff != "" {
		t.Errorf("the defaults are no longer the policy that was verified:\n%s\n"+
			"Nothing can declare a policy yet — this is where that surface becomes needed.", diff)
	}
}

// dec reads a decimal the record carries as text, so a value too large for
// int64 survives it.
func dec(t *testing.T, s string) *big.Int {
	t.Helper()
	if s == "" {
		return nil
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		t.Fatalf("the record holds %q where a decimal belongs", s)
	}
	return n
}

// envDiff reports the fields two policies disagree on, as JSON so a big.Int
// reads as its number.
func envDiff(a, b poa.Env) string {
	ja, _ := json.MarshalIndent(a, "", "  ")
	jb, _ := json.MarshalIndent(b, "", "  ")
	if string(ja) == string(jb) {
		return ""
	}
	return "verified:\n" + string(ja) + "\ndefault:\n" + string(jb)
}
