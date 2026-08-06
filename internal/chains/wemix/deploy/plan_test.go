package deploy

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func planCluster() *Cluster {
	return &Cluster{
		RPCPort:        8601,
		WSPort:         8701,
		WemixBinary:    "/r/gwemix3",
		WbftBinary:     "/r/gwemix4",
		CroissantBlock: 100,
		GenesisFile:    "/r/genesis.json",
		Servers: []Server{
			{Index: 1, Host: "10.0.0.1", Role: RoleWemixBP},
			{Index: 2, Host: "10.0.0.2", Role: RoleWbftBP},
			{Index: 3, Host: "10.0.0.3", Role: RoleEndpoint, SyncMode: "snap"},
			{Index: 4, Host: "10.0.0.4", Role: RoleBootnode},
		},
	}
}

func TestNodeRole(t *testing.T) {
	cases := map[Role]node.Role{
		RoleWemixBP:  node.RoleValidator,
		RoleWbftBP:   node.RoleValidator,
		RoleEndpoint: node.RoleEndpoint,
		RoleBootnode: node.RoleBoot,
	}
	for r, want := range cases {
		if got := NodeRole(r); got != want {
			t.Errorf("NodeRole(%s) = %s, want %s", r, got, want)
		}
	}
}

func TestBuildNodeSpec_BinaryByRole(t *testing.T) {
	c := planCluster()
	prod, _ := c.ServerByIndex(1)
	val, _ := c.ServerByIndex(2)
	if got := BuildNodeSpec(c, prod, nil).Binary; got != "/r/gwemix3" {
		t.Errorf("wemix_bp binary = %q, want /r/gwemix3", got)
	}
	if got := BuildNodeSpec(c, val, nil).Binary; got != "/r/gwemix4" {
		t.Errorf("wbft_bp binary = %q, want /r/gwemix4", got)
	}
}

func TestBuildNodeSpec_PortsAndConfig(t *testing.T) {
	c := planCluster()
	s, _ := c.ServerByIndex(3)
	spec := BuildNodeSpec(c, s, []string{"enode://abc@10.0.0.1:30303"})
	if spec.Ports.HTTP != 8601 || spec.Ports.WS != 8701 || spec.Ports.P2P != 30303 || spec.Ports.Auth != 8603 {
		t.Errorf("ports = %+v", spec.Ports)
	}
	if spec.Host != "10.0.0.3" || spec.Role != node.RoleEndpoint {
		t.Errorf("host/role = %s/%s", spec.Host, spec.Role)
	}
	if len(spec.ConfigContent) == 0 {
		t.Error("config content empty")
	}
	cfg := string(spec.ConfigContent)
	if !strings.Contains(cfg, "wemix") { // poa RPC namespace
		t.Errorf("config missing wemix namespace:\n%s", cfg)
	}
	if !strings.Contains(cfg, "snap") { // per-server sync mode
		t.Errorf("config missing snap sync mode:\n%s", cfg)
	}
	if len(spec.Args) == 0 {
		t.Error("launch args empty")
	}
}

func TestBuildNodeSpecs_LaunchOrder(t *testing.T) {
	c := planCluster()
	specs := BuildNodeSpecs(c, nil)
	if len(specs) != 4 {
		t.Fatalf("specs = %d", len(specs))
	}
	// endpoint (3) and bootnode (4) come before producer (1) and validator (2).
	if specs[0].Role != node.RoleEndpoint && specs[0].Role != node.RoleBoot {
		t.Errorf("first launched = %s, want endpoint/boot", specs[0].Role)
	}
	last := specs[len(specs)-1].Role
	if last != node.RoleValidator {
		t.Errorf("last launched = %s, want validator", last)
	}
}

func TestDescribe(t *testing.T) {
	c := planCluster()
	out := Describe(c, BuildNodeSpecs(c, nil))
	for _, want := range []string{"deploy plan: 4 server", "croissant block 100", "10.0.0.1", "/r/gwemix4"} {
		if !strings.Contains(out, want) {
			t.Errorf("describe missing %q:\n%s", want, out)
		}
	}
}
