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

// TestRoleLabel_RoundTrip covers the alias spelling a test definition uses to
// address a node by role. The identity ("node7") is deliberately not a role
// label: "node" is not a role, and inventing one would let a typo resolve.
func TestRoleLabel_RoundTrip(t *testing.T) {
	for _, c := range []struct {
		role node.Role
		ord  int
		want netmap.NodeLabel
	}{
		{node.RoleBP, 1, "bp1"},
		{node.RoleEN, 2, "en2"},
		{node.RolePN, 10, "pn10"},
	} {
		got := netmap.RoleLabel(c.role, c.ord)
		if got != c.want {
			t.Fatalf("RoleLabel(%q, %d) = %q, want %q", c.role, c.ord, got, c.want)
		}
		role, ord, err := netmap.ParseRoleLabel(got)
		if err != nil {
			t.Fatalf("ParseRoleLabel(%q): %v", got, err)
		}
		if role != c.role || ord != c.ord {
			t.Fatalf("ParseRoleLabel(%q) = (%q, %d), want (%q, %d)", got, role, ord, c.role, c.ord)
		}
	}
}

func TestParseRoleLabel_FoldsLegacyAndRejectsTheRest(t *testing.T) {
	// A legacy spelling folds onto the canonical role, like everywhere else.
	for _, in := range []netmap.NodeLabel{"validator1", "endpoint3"} {
		role, ord, err := netmap.ParseRoleLabel(in)
		if err != nil {
			t.Fatalf("ParseRoleLabel(%q): %v", in, err)
		}
		if role != node.RoleBP && role != node.RoleEN {
			t.Fatalf("ParseRoleLabel(%q) role = %q, want a canonical role", in, role)
		}
		if ord < 1 {
			t.Fatalf("ParseRoleLabel(%q) ord = %d, want >= 1", in, ord)
		}
	}
	// Not role labels: an identity label, a role with no ordinal, a zero
	// ordinal (labels are 1-based), and a word that is not a role.
	for _, in := range []netmap.NodeLabel{"node7", "en", "en0", "xyz1", "1"} {
		if _, _, err := netmap.ParseRoleLabel(in); err == nil {
			t.Fatalf("ParseRoleLabel(%q) must error", in)
		}
	}
}

// TestPlacement_CarriesBothNames fixes the decision that a node has one
// identity and one alias: the index reaches disk, the role label addresses it.
func TestPlacement_CarriesBothNames(t *testing.T) {
	p := netmap.Placement{Index: 7, Label: netmap.LabelFor(7), Role: node.RoleEN, Ord: 2}
	if p.Label != "node7" {
		t.Fatalf("identity label = %q, want node7", p.Label)
	}
	if p.RoleLabel() != "en2" {
		t.Fatalf("role label = %q, want en2", p.RoleLabel())
	}
}
