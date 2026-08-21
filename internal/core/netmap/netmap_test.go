package netmap_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestLabelFor_RoundTrips pins the labelling convention the DSL used to
// hard-code: changing it must now be a decision here, not a drift there.
func TestLabelFor_RoundTrips(t *testing.T) {
	for _, i := range []int{1, 4, 15} {
		l := netmap.LabelFor(i)
		got, err := l.Index()
		if err != nil || got != i {
			t.Errorf("LabelFor(%d)=%q, Index()=%d,%v", i, l, got, err)
		}
	}
	for _, bad := range []netmap.NodeLabel{"faucet", "node0", "nodeX", "node"} {
		if _, err := bad.Index(); err == nil {
			t.Errorf("%q parsed as an indexed label", bad)
		}
	}
}

// TestNormalizeRole_FoldsEverySpelling is the single folding table: both the
// canonical vocabulary and the legacy spellings land on bp/en/pn, and an
// unknown word is an error rather than an invented role.
func TestNormalizeRole_FoldsEverySpelling(t *testing.T) {
	cases := []struct {
		in   string
		want node.Role
	}{
		{"bp", node.RoleBP}, {"validator", node.RoleBP},
		{"en", node.RoleEN}, {"endpoint", node.RoleEN},
		{"pn", node.RolePN},
		{"boot", node.RoleBoot}, // a role until N0 demotes it to an attribute
	}
	for _, tc := range cases {
		got, err := netmap.NormalizeRole(tc.in)
		if err != nil || got != tc.want {
			t.Errorf("NormalizeRole(%q) = %v, %v; want %v", tc.in, got, err, tc.want)
		}
	}
	if _, err := netmap.NormalizeRole("miner"); err == nil {
		t.Error("an unknown role was accepted")
	}
	// The transition mapping goes back exactly, so behaviour is unchanged
	// until NM3 flips the persisted spelling deliberately.
	if netmap.LegacySpelling(node.RoleBP) != node.RoleValidator ||
		netmap.LegacySpelling(node.RoleEN) != node.RoleEndpoint ||
		netmap.LegacySpelling(node.RolePN) != node.RolePN {
		t.Error("LegacySpelling does not mirror the folding")
	}
}

// TestMap_RejectsAddressCollisions is the point of building the map before
// anything launches: two nodes on one (host, port) cannot both start, and the
// composition should say which two rather than letting the second one die at
// bind time.
func TestMap_RejectsAddressCollisions(t *testing.T) {
	p := func(label string, host string, p2p int) netmap.Placement {
		return netmap.Placement{
			Label: netmap.NodeLabel(label), Role: node.RoleBP, Host: host,
			Ports: netmap.Ports{P2P: p2p, HTTP: p2p + 300},
		}
	}

	if _, err := netmap.NewMap([]netmap.Placement{p("node1", "10.0.0.1", 8080), p("node2", "10.0.0.1", 8080)}); err == nil {
		t.Fatal("two nodes on one host:port were accepted")
	} else if !strings.Contains(err.Error(), "node1") || !strings.Contains(err.Error(), "node2") {
		t.Errorf("the collision error should name both nodes: %v", err)
	}

	// Same port on different hosts is the remote layout and must pass.
	m, err := netmap.NewMap([]netmap.Placement{p("node1", "10.0.0.1", 8080), p("node2", "10.0.0.2", 8080)})
	if err != nil {
		t.Fatalf("same port on different hosts rejected: %v", err)
	}

	// Forward and reverse agree.
	if got, ok := m.At("10.0.0.2", 8080); !ok || got != "node2" {
		t.Errorf("At(10.0.0.2:8080) = %q, %v", got, ok)
	}
	if _, ok := m.At("10.0.0.3", 8080); ok {
		t.Error("an unassigned address resolved to a node")
	}
	if pl, ok := m.Lookup("node1"); !ok || pl.Host != "10.0.0.1" {
		t.Errorf("Lookup(node1) = %+v, %v", pl, ok)
	}
}

// TestMap_RejectsDuplicateLabels: one label, one node.
func TestMap_RejectsDuplicateLabels(t *testing.T) {
	pl := netmap.Placement{Label: "node1", Role: node.RoleBP, Host: "h", Ports: netmap.Ports{P2P: 1}}
	pl2 := pl
	pl2.Ports = netmap.Ports{P2P: 2}
	if _, err := netmap.NewMap([]netmap.Placement{pl, pl2}); err == nil {
		t.Fatal("a duplicate label was accepted")
	}
}
