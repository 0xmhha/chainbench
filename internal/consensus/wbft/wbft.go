// Package wbft implements the "wbft" consensus family shared by the stablenet
// and wbft chains (decision D9): BFT block production with BLS validators,
// consensus RPC under the "istanbul" namespace. Chain-specific parameters
// (chain id, genesis engine field, system contracts) come from the chain
// plugin, not this family.
package wbft

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// Family is the wbft consensus-family strategy.
type Family struct{}

// New returns the wbft consensus family.
func New() Family { return Family{} }

func (Family) ID() string               { return "wbft" }
func (Family) RPCNamespace() string     { return "istanbul" }
func (Family) ValidatorsMethod() string { return "istanbul_getValidators" }

// BuildGenesis substitutes the wbft-family placeholders in template with the
// family-relevant fields of params (validators/BLS/extra-data/members/alloc).
func (Family) BuildGenesis(template []byte, p registry.GenesisParams) ([]byte, error) {
	return BuildGenesis(template, GenesisParams{
		ChainID:    p.ChainID,
		Validators: p.Validators,
		BLSKeys:    p.BLSKeys,
		ExtraData:  p.ExtraData,
		Members:    p.Members,
		Alloc:      p.Alloc,
	})
}

// StartFlags returns the node launch flags for a role. Validators mine; all
// nodes allow the dev-oriented RPC surface chainbench relies on.
func (Family) StartFlags(role node.Role) []string {
	flags := []string{
		"--allow-insecure-unlock",
		"--rpc.enabledeprecatedpersonal",
		"--rpc.allow-unprotected-txs",
	}
	if role == node.RoleValidator {
		flags = append(flags, "--mine")
	}
	return flags
}
