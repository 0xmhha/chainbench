package preset

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Chain is one golden chain preset (presets/chain/*.yaml) decoded.
//
// Presets come in two families, and the name says which this is. A CHAIN preset
// declares how a network is configured; a KEY preset declares the identities it
// runs as ([Key], read from [KeysDir]). presets/chain is one kind; the
// directory layout is presets/<kind>/ and more may follow.
//
// What is still read is narrower than what the document holds, and the gap is
// worth stating rather than discovering. Production reads Upgrade.AtFork and
// Upgrade.ForkBlock and nothing else. Roles, Identities, Producers and
// Validators are read only by tests that hold this document to the key set it
// describes. Chains, Data, Ports and Name are read by nothing, and the yaml's
// "description" and "nodes" keys have no field here at all, so whatever they
// say is decoded into nothing.
//
// The reason is history, not oversight: those sections were the hardfork
// handoff composer's inputs, and absorbing that composer into the ordinary
// composition path (#419) left the composition taking the same facts from the
// topology, the chain declaration, the server set and the key set. The document
// kept them. See design-v3/cohesion-candidates-2026-09-21.md §B.
//
// It was called Profile, which named neither family nor kind, and matched
// neither the directory it lives in nor the DSL field that selects it.
type Chain struct {
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

// LoadChainPreset reads and decodes a golden chain preset.
func LoadChainPreset(path string) (Chain, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Chain{}, fmt.Errorf("preset: read chain preset: %w", err)
	}
	var p Chain
	if err := yaml.Unmarshal(b, &p); err != nil {
		return Chain{}, fmt.Errorf("preset: parse chain preset %s: %w", path, err)
	}
	return p, nil
}
