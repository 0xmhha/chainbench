package poa

import (
	"encoding/json"
	"github.com/0xmhha/chainbench/internal/resource"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

const wemixTmpl = `{"config":{"chainId":__CHAIN_ID__,"istanbulBlock":0},"coinbase":"__COINBASE__","alloc":{}}`

func TestPrepareTemplate(t *testing.T) {
	out, err := PrepareTemplate([]byte(wemixTmpl), 8285, "0xb4388353fd0f3b3a017e09f2b857052ff219e663")
	if err != nil {
		t.Fatalf("BuildGenesis: %v", err)
	}
	var g struct {
		Config struct {
			ChainID int64 `json:"chainId"`
		} `json:"config"`
		Coinbase string `json:"coinbase"`
	}
	if err := json.Unmarshal(out, &g); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if g.Config.ChainID != 8285 || g.Coinbase != "0xb4388353fd0f3b3a017e09f2b857052ff219e663" {
		t.Errorf("got chainId=%d coinbase=%s", g.Config.ChainID, g.Coinbase)
	}
	if strings.Contains(string(out), "__") {
		t.Errorf("unsubstituted placeholder: %s", out)
	}
}

func TestPrepareTemplate_DefaultCoinbase(t *testing.T) {
	out, err := PrepareTemplate([]byte(wemixTmpl), 8285, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), zeroAddress) {
		t.Errorf("empty coinbase should default to zero address:\n%s", out)
	}
}

func TestPrepareTemplate_InvalidChainID(t *testing.T) {
	if _, err := PrepareTemplate([]byte(wemixTmpl), 0, ""); err == nil {
		t.Error("expected error for chainId 0")
	}
}

func TestBootstrapPlan_Order(t *testing.T) {
	plan := BootstrapPlan()
	want := []string{"init-boot", "start-boot", "deploy-governance", "init-etcd", "start-nodes"}
	if len(plan) != len(want) {
		t.Fatalf("plan length %d, want %d", len(plan), len(want))
	}
	for i, s := range plan {
		if s.Name != want[i] {
			t.Errorf("step %d: got %q, want %q", i, s.Name, want[i])
		}
	}
	// governance/etcd steps run on the boot node.
	if !plan[2].OnBootNode || !plan[3].OnBootNode {
		t.Error("deploy-governance and init-etcd must run on the boot node")
	}
}

func TestFamily_StaticFacts(t *testing.T) {
	f := New()
	if f.ID() != "poa" || f.RPCNamespace() != "wemix" || f.ValidatorsMethod() != "wemix_getValidators" {
		t.Errorf("poa family facts wrong: %s/%s/%s", f.ID(), f.RPCNamespace(), f.ValidatorsMethod())
	}
	if !BootRole(node.RoleBoot) || BootRole(node.RoleEndpoint) {
		t.Error("BootRole classification wrong")
	}
}

// TestSupportsRole_PoaHasNoProxyTier: etcd occupies that place, so a pn here is
// a declaration that would never be honoured.
func TestSupportsRole_PoaHasNoProxyTier(t *testing.T) {
	f := Family{}
	if f.SupportsRole(node.RolePN) {
		t.Error("poa must not claim a proxy tier")
	}
	for _, role := range []node.Role{node.RoleBP, node.RoleValidator, node.RoleEN, node.RoleBoot} {
		if !f.SupportsRole(role) {
			t.Errorf("poa should run %q", role)
		}
	}
}

// TestStartFlags_MineFollowsTheRoleNotItsSpelling mirrors the wbft case: the
// producer and the bootstrap node both seal, under either spelling.
func TestStartFlags_MineFollowsTheRoleNotItsSpelling(t *testing.T) {
	f := Family{}
	for _, role := range []node.Role{node.RoleBP, node.RoleValidator, node.RoleBoot} {
		found := false
		for _, fl := range f.StartFlags(role) {
			if fl == "--mine" {
				found = true
			}
		}
		if !found {
			t.Fatalf("role %q must seal: %v", role, f.StartFlags(role))
		}
	}
}

// TestPortReservation_LeavesRoomForBothEtcdPorts is the rule the previous
// global check got wrong. A wemix node's embedded etcd listens on p2p+1 (peer)
// and p2p+2 (client), so three consecutive ports belong to one node. The old
// rule allowed a step of two, which put the next node's p2p port on this node's
// etcd client — a bind failure that stalls block production with no obvious
// cause, which is the failure the rule was written to prevent in the first
// place.
func TestPortReservation_LeavesRoomForBothEtcdPorts(t *testing.T) {
	res := Family{}.PortReservation()
	if res.P2PSpan != 3 {
		t.Fatalf("p2p span = %d, want 3 (p2p, etcd peer, etcd client)", res.P2PSpan)
	}
	// A step of two is refused, and a plan with room assigns both etcd ports.
	if _, err := resource.Plan(1, 30010, 2, 40010, 10, res); err == nil {
		t.Fatal("a step of two is one short for an etcd family")
	}
	p, err := resource.Plan(1, 30010, 10, 40010, 10, res)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if p.Etcd != p.P2P+1 || p.EtcdClient != p.P2P+2 {
		t.Fatalf("etcd ports = peer %d client %d, want %d/%d", p.Etcd, p.EtcdClient, p.P2P+1, p.P2P+2)
	}
}

// TestBringUpPhases_BootLastThenJoinOneAtATime is the order that makes a wemix
// network possible at all: the etcd cluster forms while the boot node is alone,
// and the remaining producers join it one at a time. Started together, the
// cluster stays empty and the chain never produces; started all at once the
// joiners race to seal competing blocks. The boot node is the last (highest)
// producer, and the rest join highest-first; endpoints come up last.
func TestBringUpPhases_BootLastThenJoinOneAtATime(t *testing.T) {
	phases := Family{}.BringUpPhases([]node.Role{node.RoleBP, node.RoleBP, node.RoleEN})
	if len(phases) != 3 {
		t.Fatalf("phases = %d, want boot + one join + endpoints", len(phases))
	}
	boot := phases[0]
	if len(boot.Nodes) != 1 || boot.Nodes[0] != 2 {
		t.Fatalf("boot phase = %v, want only the last producer (node2)", boot.Nodes)
	}
	// The init's return value proves nothing, so the verification is part of
	// the phase rather than left to whoever wires it.
	want := []string{ActionDeployGovernance, ActionEtcdInit, ActionVerifyEtcd}
	if len(boot.Actions) != len(want) {
		t.Fatalf("boot actions = %v, want %v", boot.Actions, want)
	}
	for i, a := range want {
		if boot.Actions[i] != a {
			t.Fatalf("boot actions = %v, want %v", boot.Actions, want)
		}
	}
	join := phases[1]
	if len(join.Nodes) != 1 || join.Nodes[0] != 1 {
		t.Fatalf("join phase = %v, want the one remaining producer (node1)", join.Nodes)
	}
	// The endpoints phase seals nothing and carries no bootstrap action.
	if end := phases[2]; len(end.Nodes) != 1 || end.Nodes[0] != 3 || len(end.Actions) != 0 {
		t.Fatalf("endpoints phase = %+v, want node3 with no actions", end)
	}
}

// TestBringUpPhases_EachRemainingProducerJoinsOnItsOwnPhase: a producer that is
// not in the etcd cluster never takes a turn at sealing, so each join is its own
// phase (start then join), carried on the joining node so only it is syncing.
func TestBringUpPhases_EachRemainingProducerJoinsOnItsOwnPhase(t *testing.T) {
	phases := Family{}.BringUpPhases([]node.Role{node.RoleBP, node.RoleBP, node.RoleBP, node.RoleEN})
	// boot(node3) + join(node2) + join(node1) + endpoints(node4).
	if len(phases) != 4 {
		t.Fatalf("phases = %d, want boot + two joins + endpoints", len(phases))
	}
	if phases[0].Nodes[0] != 3 {
		t.Fatalf("boot = node%d, want the highest producer node3", phases[0].Nodes[0])
	}
	// Joins descend from the highest remaining producer.
	for i, wantNode := range []int{2, 1} {
		p := phases[i+1]
		if len(p.Nodes) != 1 || p.Nodes[0] != wantNode {
			t.Fatalf("join phase %d = %v, want [node%d]", i, p.Nodes, wantNode)
		}
		if len(p.Actions) != 1 || p.Actions[0] != ActionEtcdJoin {
			t.Fatalf("join phase %d actions = %v, want [%s]", i, p.Actions, ActionEtcdJoin)
		}
		// The join is carried on the joining node — it runs there and syncs there.
		if p.ActionsOn != wantNode {
			t.Fatalf("join phase %d runs on node%d, want the joining node%d", i, p.ActionsOn, wantNode)
		}
	}
}

// TestBringUpPhases_LoneProducerHasNothingToJoin: one producer is a cluster of
// one, already verified by the boot phase. There is no rest phase to carry a
// join, and inventing one would ask a node to join itself.
func TestBringUpPhases_LoneProducerHasNothingToJoin(t *testing.T) {
	phases := Family{}.BringUpPhases([]node.Role{node.RoleBP})
	if len(phases) != 1 {
		t.Fatalf("phases = %d, want the boot phase alone", len(phases))
	}
}
