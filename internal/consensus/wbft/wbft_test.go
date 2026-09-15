package wbft_test

import (
	"github.com/0xmhha/chainbench/internal/resource"
	"testing"

	wbft "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestLaunchPolicy_OnlyAProducerSeals: sealing was once gated on one spelling
// of the producing role, so a producer recorded under the other word launched
// without --mine and the chain stalled while every node reported healthy.
//
// The policy carries the fact and not the flag. Which flag says "seal", and
// whether the binary has it, is the dialect's question.
func TestLaunchPolicy_OnlyAProducerSeals(t *testing.T) {
	f := wbft.New()
	if !f.LaunchPolicy(node.RoleBP).Mine {
		t.Error("a bp must seal")
	}
	for _, role := range []node.Role{node.RoleEN, node.RolePN, node.Role("sideways")} {
		if f.LaunchPolicy(role).Mine {
			t.Errorf("role %q must not seal", role)
		}
	}
}

func TestSupportsRole_WbftHasAProxyTier(t *testing.T) {
	f := wbft.New()
	for _, role := range []node.Role{node.RoleBP, node.RoleEN, node.RolePN} {
		if !f.SupportsRole(role) {
			t.Errorf("wbft should run %q", role)
		}
	}
	// A word outside the vocabulary is refused rather than folded onto the role
	// it resembles. "validator" is what a bp does while another bp proposes,
	// and "boot" was a way to say pn; neither is a role a family can run.
	for _, retired := range []node.Role{"validator", "endpoint", "boot", "sideways"} {
		if f.SupportsRole(retired) {
			t.Errorf("wbft should not claim %q", retired)
		}
	}
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
	tight, err := resource.Plan(4, 30301, 1, 8601, 4, res)
	if err != nil {
		t.Fatalf("Plan(tight): %v", err)
	}
	if tight.P2P != 30304 || tight.Etcd != 0 {
		t.Fatalf("tight plan = %+v, want p2p 30304 and no etcd", tight)
	}
	// ...and the historical spacing still yields the same ports it always did.
	p, err := resource.Plan(1, 31000, 10, 8600, 10, res)
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
