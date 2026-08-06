package poa

import (
	"encoding/json"
	"fmt"
	"math/big"
)

// Config is the wemix governance/genesis config JSON consumed by
// `gwemix wemix genesis` (to materialize the genesis) and by
// `gwemix wemix deploy-governance` (to deploy the contracts). The SAME file is
// used for both. Members here are the wemix+etcd PRODUCER nodes only — a
// go-wbft validator listed as a member registers in governance but never joins
// etcd (it runs wbft), shows as "down", and stalls the producer; keep it out.
type Config struct {
	ExtraData    string    `json:"extraData"`
	Staker       string    `json:"staker"`
	Ecosystem    string    `json:"ecosystem"`
	Maintenance  string    `json:"maintenance"`
	FeeCollector string    `json:"feecollector"`
	Env          Env       `json:"env"`
	Members      []Member  `json:"members"`
	Accounts     []Account `json:"accounts"`
}

// Env is the governance environment (policy) block. Large values are big.Int so
// they marshal as JSON numbers (which gwemix expects), not strings.
type Env struct {
	BallotDurationMin    int64    `json:"ballotDurationMin"`
	BallotDurationMax    int64    `json:"ballotDurationMax"`
	StakingMin           *big.Int `json:"stakingMin"`
	StakingMax           *big.Int `json:"stakingMax"`
	MaxIdleBlockInterval int64    `json:"MaxIdleBlockInterval"`
	BlockCreationTime    int64    `json:"blockCreationTime"`
	BlockRewardAmount    *big.Int `json:"blockRewardAmount"`
	MaxPriorityFeePerGas *big.Int `json:"maxPriorityFeePerGas"`
	RewardDistribution   []int    `json:"rewardDistributionMethod"`
	MaxBaseFee           *big.Int `json:"maxBaseFee"`
	BlockGasLimit        int64    `json:"blockGasLimit"`
	BaseFeeMaxChangeRate int64    `json:"baseFeeMaxChangeRate"`
	GasTargetPercentage  int64    `json:"gasTargetPercentage"`
}

// Member is one wemix producer node. ID is the 128-hex idv5 node id (not the
// 64-hex idv4); Port is the node's p2p port (the enode go-wemix auto-dials).
type Member struct {
	Addr     string   `json:"addr"`
	Stake    *big.Int `json:"stake"`
	Name     string   `json:"name"`
	ID       string   `json:"id"`
	IP       string   `json:"ip"`
	Port     int      `json:"port"`
	Bootnode bool     `json:"bootnode"`
}

// Account is a genesis pre-funded account (for gas / governance deploy).
type Account struct {
	Addr    string   `json:"addr"`
	Balance *big.Int `json:"balance"`
}

// JSON marshals the config to the gwemix wemix-config shape.
func (c Config) JSON() ([]byte, error) { return json.MarshalIndent(c, "", " ") }

// Validate enforces the invariants that make a wemix bring-up work: at least
// one member, exactly one bootnode, and each member carrying an id/addr/port.
func (c Config) Validate() error {
	if len(c.Members) == 0 {
		return fmt.Errorf("poa: wemix config has no members")
	}
	boots := 0
	for i, m := range c.Members {
		if m.Addr == "" || m.ID == "" || m.Port == 0 {
			return fmt.Errorf("poa: member %d missing addr/id/port", i)
		}
		if len(m.ID) != 2+128 { // "0x" + 128 hex (idv5)
			return fmt.Errorf("poa: member %d id must be a 0x + 128-hex idv5 node id, got len %d", i, len(m.ID))
		}
		if m.Bootnode {
			boots++
		}
	}
	if boots != 1 {
		return fmt.Errorf("poa: wemix config needs exactly one bootnode member, got %d", boots)
	}
	return nil
}
