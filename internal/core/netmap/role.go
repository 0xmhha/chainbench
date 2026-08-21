package netmap

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// NormalizeRole folds a role spelling onto the canonical vocabulary
// (bp / en / pn, worklist N0). The legacy spellings keep working because they
// are written into persisted workspaces and topology files; unknown spellings
// are an error rather than a silently-invented role.
//
// This is the one place the folding lives. Before it, topology kept its own
// alias table and every consumer compared against whichever spelling it
// happened to know.
func NormalizeRole(s string) (node.Role, error) {
	switch node.Role(s) {
	case node.RoleBP, node.RoleValidator:
		return node.RoleBP, nil
	case node.RoleEN, node.RoleEndpoint:
		return node.RoleEN, nil
	case node.RolePN:
		return node.RolePN, nil
	case node.RoleBoot:
		// Still a role until the poa bring-up treats boot as an attribute (N0).
		return node.RoleBoot, nil
	default:
		return "", fmt.Errorf("netmap: unknown role %q (want bp, en, pn, or a legacy spelling)", s)
	}
}

// LegacySpelling maps a canonical role back to the spelling persisted state
// and the launch flows still carry ("bp" → "validator").
//
// It exists only for the transition: the composition still writes and compares
// the legacy words, and flipping them is NM3's job because it changes argv and
// persisted workspaces and needs the live re-verification that goes with that.
// When NM3 lands, this function goes with it.
func LegacySpelling(r node.Role) node.Role {
	switch r {
	case node.RoleBP:
		return node.RoleValidator
	case node.RoleEN:
		return node.RoleEndpoint
	default:
		return r
	}
}
