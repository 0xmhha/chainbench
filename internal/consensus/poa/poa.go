// Package poa implements the "poa" consensus family used by the wemix chain
// (decision D9): etcd-based membership with governance contracts deployed by a
// boot node, consensus RPC under the "wemix" namespace. The etcd bootstrap
// sequence (boot node -> deploy governance -> init etcd -> start others, per
// ../script/wemix-upgrade) attaches to Bootstrap when the setup pipeline lands
// in phase G7; G0 provides the static, RPC-facing surface.
package poa

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// Family is the poa (wemix/etcd) consensus-family strategy.
type Family struct{}

// New returns the poa consensus family.
func New() Family { return Family{} }

func (Family) ID() string               { return "poa" }
func (Family) RPCNamespace() string     { return "wemix" }
func (Family) ValidatorsMethod() string { return "wemix_getValidators" }

// BuildGenesis renders the poa base genesis: the validator set is not in the
// genesis (membership is set at bootstrap), so only ChainID and Coinbase apply.
func (Family) BuildGenesis(template []byte, p registry.GenesisParams) ([]byte, error) {
	return BuildGenesis(template, p.ChainID, p.Coinbase)
}

// StartFlags returns the node launch flags for a role. The boot node performs
// governance deployment and etcd initialization out of band.
func (Family) StartFlags(role node.Role) []string {
	flags := []string{
		"--allow-insecure-unlock",
	}
	if role == node.RoleValidator || role == node.RoleBoot {
		flags = append(flags, "--mine")
	}
	return flags
}
