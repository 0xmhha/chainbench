package upgrade

import (
	"fmt"
	"math/big"
	"os"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/core/genesis"
)

// Profile is the golden upgrade profile (profiles/wemix-upgrade.yaml) decoded.
// It is the single, declarative record of the environment under test: every
// value BuildPlan needs comes from here, so there are no code defaults to hide
// what was actually run.
type Profile struct {
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
type ChainBinding struct {
	Binary     string `yaml:"binary"`
	BinaryPath string `yaml:"binary_path"`
	NodekeyDir string `yaml:"nodekey_dir"`
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
// identity order (plan node k = preset node k) when the profile omits it.
func (p Profile) PlanOrderOrDefault() []int {
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

// LoadProfile reads and decodes a golden upgrade profile.
func LoadProfile(path string) (Profile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, fmt.Errorf("upgrade: read profile: %w", err)
	}
	var p Profile
	if err := yaml.Unmarshal(b, &p); err != nil {
		return Profile{}, fmt.Errorf("upgrade: parse profile %s: %w", path, err)
	}
	return p, nil
}

// Inputs maps the profile to BuildPlan Inputs. fromGenesis is the from-chain's
// base genesis (a bootstrap artifact, produced out-of-band), which the profile
// deliberately does not carry. It validates that the profile is internally
// complete rather than silently defaulting anything.
func (p Profile) Inputs(fromGenesis []byte) (Inputs, error) {
	if p.Upgrade.ForkBlock < 0 {
		return Inputs{}, fmt.Errorf("upgrade profile: fork_block must be >= 0")
	}
	if p.Upgrade.NetworkID <= 0 {
		return Inputs{}, fmt.Errorf("upgrade profile: network_id must be set")
	}
	if len(p.Validators.Addresses) != len(p.Validators.BLSPublicKeys) {
		return Inputs{}, fmt.Errorf("upgrade profile: %d validator addresses but %d bls keys",
			len(p.Validators.Addresses), len(p.Validators.BLSPublicKeys))
	}
	if len(p.Producers.Members) == 0 {
		return Inputs{}, fmt.Errorf("upgrade profile: no producer members")
	}
	return Inputs{
		Roles:       Roles{Producers: p.Roles.Producers, Validators: p.Roles.Validators},
		NetworkID:   p.Upgrade.NetworkID,
		ForkBlock:   big.NewInt(p.Upgrade.ForkBlock),
		FromGenesis: fromGenesis,
		ToGenesis: genesis.Inputs{
			Validators: p.Validators.Addresses,
			BLSKeys:    p.Validators.BLSPublicKeys,
			Members:    p.Validators.Members,
			ExtraData:  p.Validators.ExtraData,
		},
		ProducerAddrs: p.Producers.Members,
		P2PBase:       p.Ports.BaseP2P,
		P2PStep:       p.Ports.StepP2P,
		RPCBase:       p.Ports.BaseRPC,
		RPCStep:       p.Ports.StepRPC,
	}, nil
}
