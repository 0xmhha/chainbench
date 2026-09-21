package poa

import (
	"math/big"

	"github.com/0xmhha/chainbench/internal/preset"
)

// EnvFromPreset is the governance policy a chain preset declares, in the terms
// this genesis source takes.
//
// It lives here rather than on the preset because the preset OWNS the
// declaration and this package owns what the declaration means to poa. Putting
// the adapter on the document made the document import a consensus family, and
// a document that has to know its readers is not a document.
func EnvFromPreset(g preset.Governance) Env {
	return Env{
		BallotDurationMin: g.BallotDurationMin, BallotDurationMax: g.BallotDurationMax,
		StakingMin: presetDec(g.StakingMin), StakingMax: presetDec(g.StakingMax),
		MaxIdleBlockInterval: g.MaxIdleBlockInterval, BlockCreationTime: g.BlockCreationTime,
		BlockRewardAmount: presetDec(g.BlockRewardAmount), MaxPriorityFeePerGas: presetDec(g.MaxPriorityFeePerGas),
		RewardDistribution: g.RewardDistribution, MaxBaseFee: presetDec(g.MaxBaseFee),
		BlockGasLimit: g.BlockGasLimit, BaseFeeMaxChangeRate: g.BaseFeeMaxChangeRate,
		GasTargetPercentage: g.GasTargetPercentage,
	}
}

// presetDec reads a decimal string the preset carries as text, so a value too
// large for int64 survives the document.
func presetDec(s string) *big.Int {
	if s == "" {
		return nil
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil
	}
	return n
}
