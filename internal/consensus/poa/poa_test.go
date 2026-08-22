package poa

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

const wemixTmpl = `{"config":{"chainId":__CHAIN_ID__,"istanbulBlock":0},"coinbase":"__COINBASE__","alloc":{}}`

func TestBuildGenesis(t *testing.T) {
	out, err := BuildGenesis([]byte(wemixTmpl), 8285, "0xb4388353fd0f3b3a017e09f2b857052ff219e663")
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

func TestBuildGenesis_DefaultCoinbase(t *testing.T) {
	out, err := BuildGenesis([]byte(wemixTmpl), 8285, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), zeroAddress) {
		t.Errorf("empty coinbase should default to zero address:\n%s", out)
	}
}

func TestBuildGenesis_InvalidChainID(t *testing.T) {
	if _, err := BuildGenesis([]byte(wemixTmpl), 0, ""); err == nil {
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
