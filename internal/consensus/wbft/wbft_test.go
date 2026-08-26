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

// TestPortReservation_IsHonest: a wbft node listens on one p2p-side port and
// reserves one. The span said 2 out of inertia until the Wemix3.5 test-server
// scheme (p2p packed one apart, 30301..30304) showed the over-reservation
// rejecting a real deployment. Existing sets keep their spacing regardless —
// ports come from the configured bands; the span only sets the minimum — and
// a wbft plan derives no etcd port, so nothing advertises a port nobody
// listens on.
func TestPortReservation_IsHonest(t *testing.T) {
	res := wbft.New().PortReservation()
	if res.P2PSpan != 1 || res.RPCSpan != 3 {
		t.Fatalf("reservation = %+v, want {1, 3}", res)
	}
	// The tight real-server scheme is accepted...
	tight, err := portplan.Plan(4, 30301, 1, 8601, 4, res)
	if err != nil {
		t.Fatalf("Plan(tight): %v", err)
	}
	if tight.P2P != 30304 || tight.Etcd != 0 {
		t.Fatalf("tight plan = %+v, want p2p 30304 and no etcd", tight)
	}
	// ...and the historical spacing still yields the same ports it always did.
	p, err := portplan.Plan(1, 31000, 10, 8600, 10, res)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if p.P2P != 31000 || p.HTTP != 8600 || p.WS != 8601 || p.Auth != 8602 {
		t.Fatalf("historical plan moved: %+v", p)
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
