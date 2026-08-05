package deploy

import (
	"fmt"
	"math/big"
	"os"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
)

// Accounts is the validator/producer/test-account material (the gitignored
// `accounts` file). Producers are the wemix governance members; validators are
// the post-fork wbft set. Most fields can be auto-filled by `remote keys read`.
type Accounts struct {
	KeystoreDir          string       `yaml:"keystore_dir"`
	KeystorePasswordFile string       `yaml:"keystore_password_file"`
	Producers            []NodeAcct   `yaml:"producers"`
	Validators           []NodeAcct   `yaml:"validators"`
	Funded               []FundedAcct `yaml:"funded,omitempty"`
}

// NodeAcct is one node's account material, keyed by server index.
type NodeAcct struct {
	Server   int    `yaml:"server"`
	Addr     string `yaml:"addr"`
	Operator string `yaml:"operator,omitempty"`
	NodeID   string `yaml:"node_id,omitempty"` // idv5 pubkey, used as the wemix Member ID
	BLS      string `yaml:"bls,omitempty"`
	BLSPoP   string `yaml:"bls_pop,omitempty"`
	Stake    string `yaml:"stake,omitempty"` // wei
}

// FundedAcct is an extra genesis-funded account.
type FundedAcct struct {
	Addr    string `yaml:"addr"`
	Balance string `yaml:"balance,omitempty"` // wei; empty -> default
}

// LoadAccounts reads the accounts file.
func LoadAccounts(path string) (*Accounts, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("deploy: read accounts: %w", err)
	}
	var a Accounts
	if err := yaml.Unmarshal(b, &a); err != nil {
		return nil, fmt.Errorf("deploy: parse accounts: %w", err)
	}
	return &a, nil
}

// ProducerAddrs returns the wemix producer coinbase addresses (to exclude when
// confirming the hardfork handoff).
func (a *Accounts) ProducerAddrs() []string {
	var out []string
	for _, p := range a.Producers {
		if p.Addr != "" {
			out = append(out, p.Addr)
		}
	}
	return out
}

// defaultBalance is the genesis balance for members/validators (1e27 wei).
var defaultBalance, _ = new(big.Int).SetString("1000000000000000000000000000", 10)

// BuildWemixConfig assembles the wemix governance config from the cluster
// producers + the accounts. Each producer server needs a matching entry in
// accounts.producers (addr/node_id/stake); the first producer is the bootnode
// and the staker. Validators and any funded accounts are genesis-funded.
func BuildWemixConfig(c *Cluster, a *Accounts) (poa.Config, error) {
	prods := c.Producers()
	if len(prods) == 0 {
		return poa.Config{}, fmt.Errorf("deploy: cluster has no wemix_bp producers")
	}
	byServer := map[int]NodeAcct{}
	for _, p := range a.Producers {
		byServer[p.Server] = p
	}

	var members []poa.Member
	var funded []poa.Account
	staker := ""
	for i, s := range prods {
		pa, ok := byServer[s.Index]
		if !ok || pa.Addr == "" {
			return poa.Config{}, fmt.Errorf("deploy: no producer account (addr) for server %d", s.Index)
		}
		if pa.NodeID == "" {
			return poa.Config{}, fmt.Errorf("deploy: producer server %d missing node_id (idv5 pubkey)", s.Index)
		}
		stake, ok := new(big.Int).SetString(strDefault(pa.Stake, "2000000000000000000000000000"), 10)
		if !ok {
			return poa.Config{}, fmt.Errorf("deploy: producer server %d invalid stake %q", s.Index, pa.Stake)
		}
		members = append(members, poa.Member{
			Addr:     pa.Addr,
			Stake:    stake,
			Name:     fmt.Sprintf("producer%d", s.Index),
			ID:       ensure0x(pa.NodeID),
			IP:       s.Host,
			Port:     c.ports().P2P,
			Bootnode: i == 0,
		})
		if i == 0 {
			staker = pa.Addr
		}
		funded = append(funded, poa.Account{Addr: pa.Addr, Balance: defaultBalance})
	}
	for _, v := range a.Validators {
		if v.Addr != "" {
			funded = append(funded, poa.Account{Addr: v.Addr, Balance: defaultBalance})
		}
	}
	for _, f := range a.Funded {
		bal := defaultBalance
		if f.Balance != "" {
			if b, ok := new(big.Int).SetString(f.Balance, 10); ok {
				bal = b
			}
		}
		funded = append(funded, poa.Account{Addr: f.Addr, Balance: bal})
	}

	cfg := poa.Config{
		ExtraData:    "chainbench remote wemix deploy",
		Staker:       staker,
		Ecosystem:    staker,
		Maintenance:  staker,
		FeeCollector: staker,
		Env:          defaultEnv(c),
		Members:      members,
		Accounts:     funded,
	}
	return cfg, cfg.Validate()
}

func defaultEnv(c *Cluster) poa.Env {
	bd := func(s string) *big.Int { v, _ := new(big.Int).SetString(s, 10); return v }
	return poa.Env{
		BallotDurationMin:    86400,
		BallotDurationMax:    604800,
		StakingMin:           bd("1500000000000000000000000"),
		StakingMax:           bd("1500000000000000000000000000"),
		MaxIdleBlockInterval: 5,
		BlockCreationTime:    1000,
		BlockRewardAmount:    bd("1000000000000000000"),
		MaxPriorityFeePerGas: bd("100000000000"),
		RewardDistribution:   []int{4000, 1000, 2500, 2500},
		MaxBaseFee:           bd("50000000000000"),
		BlockGasLimit:        105000000,
		BaseFeeMaxChangeRate: 55,
		GasTargetPercentage:  30,
	}
}

func strDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func ensure0x(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s
	}
	return "0x" + s
}
