package poa

import "math/big"

// DefaultEnv is the governance policy a standalone wemix network starts from.
//
// These are the values the verified handoff profile uses
// (profiles/wemix-upgrade.yaml). They live here rather than only in a profile
// because a network has to be composable without one: requiring an operator to
// supply thirteen governance parameters before a chain can start is how "bring
// up wemix" stayed a manual procedure. A profile or an explicit config still
// overrides them — this is the floor, not the policy.
func DefaultEnv() Env {
	dec := func(s string) *big.Int {
		n, _ := new(big.Int).SetString(s, 10)
		return n
	}
	return Env{
		BallotDurationMin:    86400,
		BallotDurationMax:    604800,
		StakingMin:           dec("1500000000000000000000000"),
		StakingMax:           dec("1500000000000000000000000000"),
		MaxIdleBlockInterval: 5,
		BlockCreationTime:    1000,
		BlockRewardAmount:    dec("1000000000000000000"),
		MaxPriorityFeePerGas: dec("100000000000"),
		RewardDistribution:   []int{4000, 1000, 2500, 2500},
		MaxBaseFee:           dec("50000000000000"),
		BlockGasLimit:        105000000,
		BaseFeeMaxChangeRate: 55,
		GasTargetPercentage:  30,
	}
}

// DefaultAccountBalance is what each declared account is funded with in a test
// network: enough that no test has to think about the faucet before it can send
// a transaction.
func DefaultAccountBalance() *big.Int {
	n, _ := new(big.Int).SetString("1000000000000000000000000000", 10)
	return n
}
