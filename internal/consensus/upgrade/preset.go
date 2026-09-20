package upgrade

import (
	"math/big"
	"strings"

	"fmt"
	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"os"

	"go.yaml.in/yaml/v3"
)

// ChainPreset is one golden chain preset (presets/<kind>/*.yaml) decoded.
//
// Presets come in two families, and the name says which this is. A CHAIN preset
// declares how a network is configured; a KEY preset declares the identities it
// runs as (keys/preset, decoded by keyring.Preset). Naming this one after the
// family rather than after the kind is deliberate: of its eleven sections only
// "upgrade" is about a hardfork, and the other ten — chains, roles, identities,
// producers, validators, data, ports, nodes — are ordinary chain configuration
// that a preset of another kind would declare the same way. presets/hardfork is
// one kind; the directory layout is presets/<kind>/ and more may follow.
//
// It is the single, declarative record of the environment under test: every
// value BuildPlan needs comes from here, so there are no code defaults to hide
// what was actually run.
//
// It was called Profile, which named neither family nor kind, and matched
// neither the directory it lives in nor the DSL field that selects it.
type ChainPreset struct {
	Name    string `yaml:"name"`
	Upgrade struct {
		From      string `yaml:"from"`
		To        string `yaml:"to"`
		AtFork    string `yaml:"at_fork"`
		ForkBlock int64  `yaml:"fork_block"`
		NetworkID int64  `yaml:"network_id"`
	} `yaml:"upgrade"`
	Chains struct {
		From ChainBinding `yaml:"from"`
		To   ChainBinding `yaml:"to"`
	} `yaml:"chains"`
	Roles struct {
		Producers  int `yaml:"producers"`
		Validators int `yaml:"validators"`
	} `yaml:"roles"`
	Producers struct {
		Members []string `yaml:"members"`
		// Stake is the producer's governance stake (wei, decimal string).
		Stake string `yaml:"stake"`
		// Governance is the wemix governance env the base genesis is built from.
		Governance Governance `yaml:"governance"`
	} `yaml:"producers"`
	// Identities maps plan nodes to preset node identities. PlanOrder[k-1] is the
	// preset node number (1-based) that supplies plan node k's key material;
	// PlanOrder[0] is the producer. Empty means plan order == preset order.
	Identities struct {
		PlanOrder []int `yaml:"plan_order"`
	} `yaml:"identities"`
	Validators struct {
		Addresses     []string `yaml:"addresses"`
		BLSPublicKeys []string `yaml:"bls_public_keys"`
		Members       []string `yaml:"members"`
		ExtraData     string   `yaml:"extra_data"`
	} `yaml:"validators"`
	Data struct {
		Directory string `yaml:"directory"`
	} `yaml:"data"`
	Ports struct {
		BaseP2P int `yaml:"base_p2p"`
		StepP2P int `yaml:"step_p2p"`
		BaseRPC int `yaml:"base_rpc"`
		StepRPC int `yaml:"step_rpc"`
	} `yaml:"ports"`
}

// ChainBinding binds one side of the handoff to a concrete binary.
//
// It used to carry nodekey_dir, the directory that side's binary looks in for
// its devp2p key. That was only ever needed because the launch wrote no config
// file and so had to put the key where each binary would find it by itself. A
// node's config names the file now, so the two binaries need not agree on a
// directory and the preset need not know either one.
type ChainBinding struct {
	Binary     string `yaml:"binary"`
	BinaryPath string `yaml:"binary_path"`
	Recommit   string `yaml:"miner_recommit"`
}

// Governance is the wemix governance env (policy parameters) the producer's base
// genesis is generated from. Large values are decimal strings; the rest are
// integers. It mirrors the fields `gwemix wemix genesis` consumes.
type Governance struct {
	BallotDurationMin    int64  `yaml:"ballot_duration_min"`
	BallotDurationMax    int64  `yaml:"ballot_duration_max"`
	StakingMin           string `yaml:"staking_min"`
	StakingMax           string `yaml:"staking_max"`
	MaxIdleBlockInterval int64  `yaml:"max_idle_block_interval"`
	BlockCreationTime    int64  `yaml:"block_creation_time"`
	BlockRewardAmount    string `yaml:"block_reward_amount"`
	MaxPriorityFeePerGas string `yaml:"max_priority_fee_per_gas"`
	RewardDistribution   []int  `yaml:"reward_distribution"`
	MaxBaseFee           string `yaml:"max_base_fee"`
	BlockGasLimit        int64  `yaml:"block_gas_limit"`
	BaseFeeMaxChangeRate int64  `yaml:"base_fee_max_change_rate"`
	GasTargetPercentage  int64  `yaml:"gas_target_percentage"`
}

// PlanOrderOrDefault returns the plan-node -> preset-node mapping, defaulting to
// identity order (plan node k = preset node k) when the preset omits it.
func (p ChainPreset) PlanOrderOrDefault() []int {
	if len(p.Identities.PlanOrder) != 0 {
		return p.Identities.PlanOrder
	}
	total := p.Roles.Producers + p.Roles.Validators
	order := make([]int, total)
	for i := range order {
		order[i] = i + 1
	}
	return order
}

// LoadChainPreset reads and decodes a golden hardfork preset.
func LoadChainPreset(path string) (ChainPreset, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return ChainPreset{}, fmt.Errorf("upgrade: read preset: %w", err)
	}
	var p ChainPreset
	if err := yaml.Unmarshal(b, &p); err != nil {
		return ChainPreset{}, fmt.Errorf("upgrade: parse preset %s: %w", path, err)
	}
	return p, nil
}

// GovernanceEnv is the governance policy the preset declares, in the terms the
// poa genesis source takes.
//
// It is here rather than on the composition because the preset is the only
// thing that has ever declared one, and because a test holds it against
// poa.DefaultEnv: as long as the two agree, a hardfork needs no governance
// declaration to compose the same chain.
func (p ChainPreset) GovernanceEnv() poa.Env {
	g := p.Producers.Governance
	return poa.Env{
		BallotDurationMin: g.BallotDurationMin, BallotDurationMax: g.BallotDurationMax,
		StakingMin: dec(g.StakingMin), StakingMax: dec(g.StakingMax),
		MaxIdleBlockInterval: g.MaxIdleBlockInterval, BlockCreationTime: g.BlockCreationTime,
		BlockRewardAmount: dec(g.BlockRewardAmount), MaxPriorityFeePerGas: dec(g.MaxPriorityFeePerGas),
		RewardDistribution: g.RewardDistribution, MaxBaseFee: dec(g.MaxBaseFee),
		BlockGasLimit: g.BlockGasLimit, BaseFeeMaxChangeRate: g.BaseFeeMaxChangeRate,
		GasTargetPercentage: g.GasTargetPercentage,
	}
}

// dec parses a decimal wei string; empty or malformed is zero.
func dec(s string) *big.Int {
	n, ok := new(big.Int).SetString(strings.TrimSpace(s), 10)
	if !ok {
		return big.NewInt(0)
	}
	return n
}
