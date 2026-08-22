package wbft_test

import (
	"testing"

	wbft "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestStartFlags_MineFollowsTheRoleNotItsSpelling: --mine was gated on the
// legacy word alone. A producer recorded with the canonical role would have
// launched without it and the chain would stall — the same latent break NM1c
// found in the selector, in the flag that decides whether blocks get made.
func TestStartFlags_MineFollowsTheRoleNotItsSpelling(t *testing.T) {
	f := wbft.New()
	for _, role := range []node.Role{node.RoleBP, node.RoleValidator} {
		if !hasFlag(f.StartFlags(role), "--mine") {
			t.Fatalf("role %q must seal: %v", role, f.StartFlags(role))
		}
	}
	for _, role := range []node.Role{node.RoleEN, node.RoleEndpoint, node.RolePN} {
		if hasFlag(f.StartFlags(role), "--mine") {
			t.Fatalf("role %q must not seal: %v", role, f.StartFlags(role))
		}
	}
}

func TestSupportsRole_WbftHasAProxyTier(t *testing.T) {
	f := wbft.New()
	for _, role := range []node.Role{node.RoleBP, node.RoleEN, node.RolePN, node.RoleValidator, node.RoleEndpoint} {
		if !f.SupportsRole(role) {
			t.Errorf("wbft should run %q", role)
		}
	}
	// No governance bootstrap in this family, and not a role at all.
	for _, role := range []node.Role{node.RoleBoot, node.Role("sideways")} {
		if f.SupportsRole(role) {
			t.Errorf("wbft should not claim %q", role)
		}
	}
}

func hasFlag(flags []string, want string) bool {
	for _, f := range flags {
		if f == want {
			return true
		}
	}
	return false
}
