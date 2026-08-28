package chainsetup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	wbftfam "github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/topology"
)

// topologyFixture writes a small stablenet topology (validators + an endpoint)
// and returns its path.
func topologyFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "topology.yaml")
	yaml := "chain: stablenet\nnetwork: local\nnodes:\n" +
		"  - { index: 1, role: bp, sync_mode: full }\n" +
		"  - { index: 2, role: bp, sync_mode: full }\n" +
		"  - { index: 3, role: en, sync_mode: archive }\n"
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatalf("write topology: %v", err)
	}
	return path
}

// localPlanPlugin is a stablenet-family plugin carrying a manifest capability so
// the capability-derivation assertions have something to preserve.
func localPlanPlugin() registry.ChainPlugin {
	return registry.StaticPlugin{
		M:   registry.Manifest{ID: "stablenet", Binary: "go-stablenet", ChainID: 8283, Capabilities: []string{"governance"}},
		Fam: wbftfam.New(),
	}
}

func hasCap(caps []string, want string) bool {
	for _, c := range caps {
		if c == want {
			return true
		}
	}
	return false
}

func TestBuildLocalPlan_PositionalRolesPortsAndCaps(t *testing.T) {
	cfg := config.Values{"nodes.validators": "2", "nodes.endpoints": "1"}
	plan, err := chainsetup.BuildLocalPlan(cfg, localPlanPlugin(), "/d", nil)
	if err != nil {
		t.Fatalf("BuildLocalPlan: %v", err)
	}
	if len(plan.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3", len(plan.Nodes))
	}
	// Nodes 1..V are validators, V+1.. are endpoints, in launch order.
	for i, want := range []node.Role{node.RoleValidator, node.RoleValidator, node.RoleEndpoint} {
		if plan.Nodes[i].Role != want {
			t.Errorf("node%d role = %q, want %q", i+1, plan.Nodes[i].Role, want)
		}
	}
	// Ports offset from the default base by zero-based index.
	if got := plan.Nodes[0].Ports.HTTP; got != 8501 {
		t.Errorf("node1 http = %d, want 8501", got)
	}
	if got := plan.Nodes[2].Ports.HTTP; got != 8503 {
		t.Errorf("node3 http = %d, want 8503 (base+2)", got)
	}
	// The launch argv is left to launcher.Direct.Arm.
	if len(plan.Nodes[0].Args) != 0 {
		t.Errorf("Args must be empty (armed by the launcher): %v", plan.Nodes[0].Args)
	}
	// Launched networks always advertise ws, and keep the manifest's own caps.
	if !hasCap(plan.Capabilities, "ws") || !hasCap(plan.Capabilities, "governance") {
		t.Errorf("caps = %v, want ws + governance", plan.Capabilities)
	}
	if hasCap(plan.Capabilities, "delayed-boho") {
		t.Errorf("delayed-boho must not be advertised without an override: %v", plan.Capabilities)
	}
}

func TestBuildLocalPlan_CustomBasePorts(t *testing.T) {
	cfg := config.Values{"nodes.validators": "1", "ports.base_http": "9000"}
	plan, err := chainsetup.BuildLocalPlan(cfg, localPlanPlugin(), "/d", nil)
	if err != nil {
		t.Fatalf("BuildLocalPlan: %v", err)
	}
	if got := plan.Nodes[0].Ports.HTTP; got != 9000 {
		t.Errorf("node1 http = %d, want the 9000 base override", got)
	}
}

func TestBuildLocalPlan_DelayedBohoAndOverlayCaps(t *testing.T) {
	cfg := config.Values{
		"nodes.validators":            "1",
		"genesis.overrides.bohoBlock": "10",
		"genesis.capabilities":        "account-extra, custom",
	}
	plan, err := chainsetup.BuildLocalPlan(cfg, localPlanPlugin(), "/d", nil)
	if err != nil {
		t.Fatalf("BuildLocalPlan: %v", err)
	}
	for _, want := range []string{"delayed-boho", "account-extra", "custom"} {
		if !hasCap(plan.Capabilities, want) {
			t.Errorf("cap %q missing: %v", want, plan.Capabilities)
		}
	}
}

func TestBuildLocalPlan_Topology(t *testing.T) {
	topo, err := topology.Load(topologyFixture(t))
	if err != nil {
		t.Fatalf("load topology: %v", err)
	}
	plan, err := chainsetup.BuildLocalPlan(config.Values{}, localPlanPlugin(), "/d", &topo)
	if err != nil {
		t.Fatalf("BuildLocalPlan: %v", err)
	}
	if len(plan.Nodes) != len(topo.Sorted()) {
		t.Fatalf("nodes = %d, want %d from topology", len(plan.Nodes), len(topo.Sorted()))
	}
	for i, tn := range topo.Sorted() {
		if plan.Nodes[i].Role != tn.NodeRole() {
			t.Errorf("node%d role = %q, want %q from topology", i+1, plan.Nodes[i].Role, tn.NodeRole())
		}
	}
}

func TestBuildLocalPlan_NoNodes(t *testing.T) {
	if _, err := chainsetup.BuildLocalPlan(config.Values{}, localPlanPlugin(), "/d", nil); err == nil {
		t.Fatal("expected an error when no nodes are configured")
	}
}
