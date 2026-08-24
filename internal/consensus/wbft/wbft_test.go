package wbft_test

import (
	"testing"

	wbft "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/portplan"
)

// TestStartFlags_MineFollowsTheRoleNotItsSpelling: --mine was gated on the
// legacy word alone. A producer recorded with the canonical role would have
// launched without it and the chain would stall — the same latent break the
// selector had, this time in the flag that decides whether blocks get made.
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

// TestPortReservation_KeepsTheSpacingNetworksAlreadyRun: wbft has no etcd, but
// the second p2p-side port stays reserved. Reclaiming it would move every
// existing network's ports for the sake of a port nobody listens on.
func TestPortReservation_KeepsTheSpacingNetworksAlreadyRun(t *testing.T) {
	res := wbft.New().PortReservation()
	if res.P2PSpan != 2 || res.RPCSpan != 3 {
		t.Fatalf("reservation = %+v, want {2, 3}", res)
	}
	p, err := portplan.Plan(1, 31000, 10, 8600, 10, res)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// The ports a composed stablenet network is running on today.
	if p.P2P != 31000 || p.HTTP != 8600 || p.WS != 8601 || p.Auth != 8602 || p.Metrics != 8603 {
		t.Fatalf("ports moved: %+v", p)
	}
	// No etcd client is reserved for a family that does not embed etcd.
	if p.EtcdClient != 0 {
		t.Fatalf("wbft should not reserve an etcd client port, got %d", p.EtcdClient)
	}
}

// TestBringUpPhases_OneGroupNoActions: the wbft families start every node at
// once. A phase naming no nodes is the whole plan, so this is the launch that
// existed before phases did.
func TestBringUpPhases_OneGroupNoActions(t *testing.T) {
	phases := wbft.New().BringUpPhases([]node.Role{node.RoleBP, node.RoleBP, node.RoleEN})
	if len(phases) != 1 {
		t.Fatalf("phases = %d, want one", len(phases))
	}
	if len(phases[0].Nodes) != 0 {
		t.Fatalf("phase names %v, want the whole plan", phases[0].Nodes)
	}
	if len(phases[0].Actions) != 0 {
		t.Fatalf("wbft needs no bring-up actions, got %v", phases[0].Actions)
	}
}
