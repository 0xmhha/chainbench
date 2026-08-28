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
	// Both spellings of the producing role seal. Comparing against one of them
	// makes --mine depend on which word the composition happened to record,
	// and a producer launched without it stalls the chain.
	if canonical, err := node.NormalizeRole(string(role)); err == nil && canonical == node.RoleBP {
		flags = append(flags, "--mine")
	}
	return flags
}

// BringUpPhases: every wbft node starts at once and nothing has to happen in
// between. One phase with no node list means the whole plan, so this is the
// launch that existed before phases did — byte for byte.
func (Family) BringUpPhases(roles []node.Role) []registry.Phase {
	return []registry.Phase{{Name: "all"}}
}

// PortReservation: wbft nodes listen on p2p, http, ws and auth — one port on
// the p2p side, nothing derived. The span used to say 2 out of inertia, and
// that over-reservation rejected a real deployment shape: the Wemix3.5 test
// servers pack p2p one apart (30301..30304) because nothing sits between.
// Existing sets keep their spacing regardless — ports come from the
// configured bands; the span only sets the minimum.
func (Family) PortReservation() node.Reservation {
	return node.Reservation{P2PSpan: 1, RPCSpan: 3}
}

// SupportsRole: the wbft family runs producers, endpoints, and a proxy tier
// between them. It has no governance bootstrap, so "boot" is not one of its
// roles.
func (Family) SupportsRole(role node.Role) bool {
	canonical, err := node.NormalizeRole(string(role))
	if err != nil {
		return false
	}
	switch canonical {
	case node.RoleBP, node.RoleEN, node.RolePN:
		return true
	default:
		return false
	}
}
