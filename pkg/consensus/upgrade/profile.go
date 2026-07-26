package upgrade

import (
	"fmt"
	"math/big"
	"os"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/pkg/core/genesis"
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
	} `yaml:"producers"`
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
