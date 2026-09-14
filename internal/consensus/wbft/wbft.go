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

// Registering here means the family is available wherever it is linked in, and
// the blank import that pulls in a chain pulls in the family the chain
// composes. Nothing has to name the set in a switch.
func init() { registry.RegisterFamily(New()) }

func (Family) ID() string { return "wbft" }

// ValidatorsCarryBLS: a wbft validator signs with a BLS key as well as its
// account key, and the genesis extra-data carries the public half.
func (Family) ValidatorsCarryBLS() bool { return true }
func (Family) RPCNamespace() string     { return "istanbul" }
func (Family) ValidatorsMethod() string { return "istanbul_getValidators" }

// wbft needs no RuntimeValidatorReader: its validators are read through the
// manifest's istanbul_getValidators method, so RunningValidators falls back to
// that. Only poa, whose validators live in a governance contract with no RPC
// method, implements a reader.

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

// LaunchPolicy says what this consensus asks of a node's launch. Only sealing:
// the dev-oriented RPC surface the harness relies on is asked for on every
// launch and granted by whichever dialect has it, which is where a binary's
// flags belong.
func (Family) LaunchPolicy(role node.Role) registry.LaunchPolicy {
	// Asking the vocabulary rather than comparing words: --mine used to depend
	// on which spelling the composition happened to record, and a producer
	// launched without it stalls the chain while every node reports healthy.
	return registry.LaunchPolicy{Mine: node.Is(role, node.RoleBP)}
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

// RingAccounts: a wbft-family chain bakes its validator set into genesis, with
// BLS material per validator, and — where the anzeon system contracts are used —
// a governance council alongside it. Both come out of the key set, so both are
// shown; a validator without BLS material is reported rather than hidden,
// because it is a set that will pass genesis and then fail to sign.
func (Family) RingAccounts(validators, blsKeys, members []string) ([]registry.RingAccount, string) {
	out := make([]registry.RingAccount, 0, len(validators)+len(members))
	for i, addr := range validators {
		detail := "no BLS"
		if i < len(blsKeys) && blsKeys[i] != "" {
			detail = "BLS present"
		}
		out = append(out, registry.RingAccount{
			Role: registry.AccountValidator, Index: i + 1, Address: addr, Detail: detail,
		})
	}
	for _, addr := range members {
		out = append(out, registry.RingAccount{
			Role: registry.AccountGovernance, Address: addr, Detail: "system-contract council",
		})
	}
	return out, ""
}
