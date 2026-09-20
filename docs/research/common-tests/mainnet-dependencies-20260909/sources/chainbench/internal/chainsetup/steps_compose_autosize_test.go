package chainsetup

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// poolOf builds a pool with n servers; placements() reads only len(Hosts).
func poolOf(n int) resource.Pool {
	hosts := make([]resource.Host, n)
	for i := range hosts {
		hosts[i] = resource.Host{Name: "server", Addr: "127.0.0.1"}
	}
	return resource.Pool{Hosts: hosts, Slots: 1}
}

// TestPlacements_AutoSizeFillsToServerCount is the dynamic-sizing contract: the
// network is one node per server, the pn is last (so it lands on the last
// server), and one en and one pn sit alongside the validators that fill the
// rest. A named count is untouched — this only fires under AutoSize.
func TestPlacements_AutoSizeFillsToServerCount(t *testing.T) {
	// 15 servers, one pn and one en by default -> 13 bp + 1 en + 1 pn, pn last.
	reqs, modes, err := AllocateOpts{AutoSize: true, Proxies: 1, Endpoints: 1, Pool: poolOf(15)}.placements()
	if err != nil {
		t.Fatalf("placements: %v", err)
	}
	if len(reqs) != 15 {
		t.Fatalf("node count = %d, want 15 (one per server)", len(reqs))
	}
	var bp, en, pn int
	for _, r := range reqs {
		switch {
		case node.Is(r.Role, node.RoleBP):
			bp++
		case node.Is(r.Role, node.RoleEN):
			en++
		case node.Is(r.Role, node.RolePN):
			pn++
		}
	}
	if bp != 13 || en != 1 || pn != 1 {
		t.Fatalf("split = %d bp / %d en / %d pn, want 13/1/1", bp, en, pn)
	}
	// The pn is the last node: with host-first slot filling that puts it on the
	// last server, which is where the model's discovery hub belongs.
	if last := reqs[len(reqs)-1]; !node.Is(last.Role, node.RolePN) {
		t.Fatalf("last node role = %q, want pn", last.Role)
	}
	if len(modes) != len(reqs) {
		t.Fatalf("modes = %d, want one per node (%d)", len(modes), len(reqs))
	}
}

// TestPlacements_AutoSizeHonoursExplicitProxyEndpointCounts: the counts the
// spec did name still hold; only the validators are filled.
func TestPlacements_AutoSizeHonoursExplicitProxyEndpointCounts(t *testing.T) {
	reqs, _, err := AllocateOpts{AutoSize: true, Proxies: 2, Endpoints: 3, Pool: poolOf(10)}.placements()
	if err != nil {
		t.Fatalf("placements: %v", err)
	}
	var bp, en, pn int
	for _, r := range reqs {
		switch {
		case node.Is(r.Role, node.RoleBP):
			bp++
		case node.Is(r.Role, node.RoleEN):
			en++
		case node.Is(r.Role, node.RolePN):
			pn++
		}
	}
	if bp != 5 || en != 3 || pn != 2 {
		t.Fatalf("split = %d bp / %d en / %d pn, want 5/3/2", bp, en, pn)
	}
}

// TestPlacements_AutoSizeNeedsAServerSet: with no pool there is no capacity to
// fill, and sizing to nothing would silently make a one-node network.
func TestPlacements_AutoSizeNeedsAServerSet(t *testing.T) {
	_, _, err := AllocateOpts{AutoSize: true, Proxies: 1, Endpoints: 1}.placements()
	if err == nil || !strings.Contains(err.Error(), "server-set") {
		t.Fatalf("err = %v, want a server-set requirement", err)
	}
}

// TestPlacements_AutoSizeRefusesAServerSetTooSmall: three servers cannot hold a
// pn, an en, and still leave a validator.
func TestPlacements_AutoSizeRefusesAServerSetTooSmall(t *testing.T) {
	_, _, err := AllocateOpts{AutoSize: true, Proxies: 1, Endpoints: 1, Pool: poolOf(2)}.placements()
	if err == nil {
		t.Fatal("a 2-server set was accepted for pn+en+bp")
	}
}

// TestPlacements_TopologyCarriesPerNodeConfig: a node table's config path rides
// the LaunchReq so the config step can write it verbatim for that node.
func TestPlacements_TopologyCarriesPerNodeConfig(t *testing.T) {
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp", Config: "/opt/cfg/node1.toml"},
		{Index: 2, Role: "en"},
	}}
	reqs, _, err := AllocateOpts{Topology: topo}.placements()
	if err != nil {
		t.Fatalf("placements: %v", err)
	}
	if len(reqs) != 2 {
		t.Fatalf("reqs = %d, want 2", len(reqs))
	}
	if reqs[0].Config != "/opt/cfg/node1.toml" {
		t.Errorf("node1 config = %q, want the pinned path", reqs[0].Config)
	}
	if reqs[1].Config != "" {
		t.Errorf("node2 config = %q, want empty", reqs[1].Config)
	}
}

// TestPlacements_NamedCountKeepsBpPnEnOrder: without AutoSize the ordering the
// existing specs address by index is unchanged (bp, then pn, then en).
func TestPlacements_NamedCountKeepsBpPnEnOrder(t *testing.T) {
	reqs, _, err := AllocateOpts{Validators: 2, Proxies: 1, Endpoints: 1}.placements()
	if err != nil {
		t.Fatalf("placements: %v", err)
	}
	want := []node.Role{node.RoleBP, node.RoleBP, node.RolePN, node.RoleEN}
	if len(reqs) != len(want) {
		t.Fatalf("node count = %d, want %d", len(reqs), len(want))
	}
	for i, w := range want {
		if !node.Is(reqs[i].Role, w) {
			t.Fatalf("node%d role = %q, want %q", i+1, reqs[i].Role, w)
		}
	}
}
