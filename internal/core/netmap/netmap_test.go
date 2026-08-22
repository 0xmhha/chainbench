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

// TestAssign_ConsumesHostsBeforeSlots is the allocation rule stated as a table:
// five addresses and four port slots hold twenty nodes, and node6 comes back to
// the first address on the next slot rather than colliding on the first.
func TestAssign_ConsumesHostsBeforeSlots(t *testing.T) {
	pool := netmap.Pool{
		Hosts: []netmap.Host{{Addr: "10.0.0.1"}, {Addr: "10.0.0.2"}, {Addr: "10.0.0.3"}, {Addr: "10.0.0.4"}, {Addr: "10.0.0.5"}},
		Slots: 4,
		Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10},
	}
	if got := pool.Cap(); got != 20 {
		t.Fatalf("Cap() = %d, want 20", got)
	}

	reqs := make([]netmap.Request, 15)
	for i := range reqs {
		reqs[i] = netmap.Request{Role: node.RoleBP}
	}
	m, err := netmap.Assign(pool, reqs)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}

	for _, c := range []struct {
		label    netmap.NodeLabel
		host     string
		p2p, rpc int
	}{
		{"node1", "10.0.0.1", 31000, 8600},
		{"node5", "10.0.0.5", 31000, 8600},
		{"node6", "10.0.0.1", 31010, 8610}, // wraps to the first host, next slot
		{"node10", "10.0.0.5", 31010, 8610},
		{"node11", "10.0.0.1", 31020, 8620},
		{"node15", "10.0.0.5", 31020, 8620},
	} {
		p, ok := m.Lookup(c.label)
		if !ok {
			t.Fatalf("%s not placed", c.label)
		}
		if p.Host != c.host || p.Ports.P2P != c.p2p || p.Ports.HTTP != c.rpc {
			t.Fatalf("%s = %s p2p=%d http=%d, want %s p2p=%d http=%d",
				c.label, p.Host, p.Ports.P2P, p.Ports.HTTP, c.host, c.p2p, c.rpc)
		}
		// The etcd port travels with the node; losing it is what NM1 fixed.
		if p.Ports.Etcd != p.Ports.P2P+1 {
			t.Fatalf("%s etcd = %d, want %d", c.label, p.Ports.Etcd, p.Ports.P2P+1)
		}
	}
}

// TestAssign_OverCapacityNamesTheShortfall: wrapping onto ports already handed
// out produces a network where two nodes cannot both bind, found much later.
func TestAssign_OverCapacityNamesTheShortfall(t *testing.T) {
	pool := netmap.Pool{
		Hosts: []netmap.Host{{Addr: "10.0.0.1"}, {Addr: "10.0.0.2"}},
		Slots: 2,
		Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10},
	}
	reqs := make([]netmap.Request, 6)
	for i := range reqs {
		reqs[i] = netmap.Request{Role: node.RoleBP}
	}
	_, err := netmap.Assign(pool, reqs)
	if err == nil {
		t.Fatal("over-capacity assignment must fail")
	}
	for _, want := range []string{"6 nodes", "2 host", "2 slot", "4", "2 short"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q should name %q", err, want)
		}
	}
}

func TestAssign_RolesAndOrdinals(t *testing.T) {
	pool := netmap.Pool{
		Hosts: []netmap.Host{{Addr: "127.0.0.1"}},
		Slots: 8,
		Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10},
	}
	// Roles interleave on purpose: a topology owns launch order (poa needs its
	// producer first), so the ordinal follows index order within a role rather
	// than requiring roles to be contiguous.
	m, err := netmap.Assign(pool, []netmap.Request{
		{Role: node.RoleBP},
		{Role: node.RoleEndpoint}, // legacy spelling folds
		{Role: node.RoleBP},
		{Role: node.RolePN},
		{Role: node.RoleEN},
	})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	want := map[netmap.NodeLabel]netmap.NodeLabel{
		"node1": "bp1", "node2": "en1", "node3": "bp2", "node4": "pn1", "node5": "en2",
	}
	for identity, alias := range want {
		p, ok := m.Lookup(identity)
		if !ok {
			t.Fatalf("%s not placed", identity)
		}
		if p.RoleLabel() != alias {
			t.Fatalf("%s alias = %q, want %q", identity, p.RoleLabel(), alias)
		}
	}
	if n := len(m.Placements()); n != 5 {
		t.Fatalf("Placements() = %d, want 5", n)
	}
}

func TestPool_ValidateRejects(t *testing.T) {
	base := netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10}
	for name, p := range map[string]netmap.Pool{
		"no hosts":     {Slots: 1, Ports: base},
		"no address":   {Hosts: []netmap.Host{{Name: "bp1"}}, Slots: 1, Ports: base},
		"no slots":     {Hosts: []netmap.Host{{Addr: "10.0.0.1"}}, Ports: base},
		"p2p step 1":   {Hosts: []netmap.Host{{Addr: "10.0.0.1"}}, Slots: 2, Ports: netmap.Bands{P2PBase: 31000, P2PStep: 1, RPCBase: 8600, RPCStep: 10}},
		"rpc step two": {Hosts: []netmap.Host{{Addr: "10.0.0.1"}}, Slots: 2, Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 2}},
	} {
		if err := p.Validate(); err == nil {
			t.Fatalf("%s: Validate must reject", name)
		}
	}
}

// TestAssign_KeepsAGivenLabel: a name an operator chose has to survive, since
// it is what they will read in a path, a log file and a test definition. The
// placement type this replaced also carried a name — invented in four spellings
// by different callers, then read by none.
func TestAssign_KeepsAGivenLabel(t *testing.T) {
	pool := netmap.Pool{
		Hosts: []netmap.Host{{Addr: "127.0.0.1"}},
		Slots: 4,
		Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10},
	}
	m, err := netmap.Assign(pool, []netmap.Request{
		{Role: node.RoleBP, Label: "seoul-bp"},
		{Role: node.RoleBP}, // unnamed: takes the conventional identity
	})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	named, ok := m.Lookup("seoul-bp")
	if !ok {
		t.Fatalf("the given label was not kept: %v", m.Labels())
	}
	if named.Index != 1 || named.RoleLabel() != "bp1" {
		t.Fatalf("a named node keeps its position and alias: %+v", named)
	}
	if _, ok := m.Lookup("node2"); !ok {
		t.Fatalf("an unnamed node takes the conventional label: %v", m.Labels())
	}
}

// TestLayout_NamesEveryPathAfterTheNode fixes the derivation that was spelled
// out with fmt.Sprintf at six call sites, each free to disagree.
func TestLayout_NamesEveryPathAfterTheNode(t *testing.T) {
	l := netmap.Layout{Root: "/srv/cb"}
	for _, c := range []struct{ got, want string }{
		{l.DataDir("bp1"), "/srv/cb/bp1"},
		{l.ConfigPath("bp1"), "/srv/cb/config_bp1.toml"},
		{l.LogPath("bp1"), "/srv/cb/logs/bp1.log"},
		{l.GenesisPath(), "/srv/cb/genesis.json"},
	} {
		if c.got != c.want {
			t.Errorf("path = %q, want %q", c.got, c.want)
		}
	}
	// The label is the only thing that varies, so two nodes never share a path.
	if l.DataDir("node1") == l.DataDir("node2") {
		t.Fatal("two nodes must not share a datadir")
	}
}

// TestAssign_IsDeterministic: the same pool and the same requests must yield
// the same map. A layout that drifts between runs would make every artifact
// derived from it — static-nodes, genesis, a saved session — disagree with the
// network it describes.
func TestAssign_IsDeterministic(t *testing.T) {
	pool := netmap.Pool{
		Hosts: []netmap.Host{{Addr: "10.0.0.1"}, {Addr: "10.0.0.2"}, {Addr: "10.0.0.3"}},
		Slots: 3,
		Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10},
	}
	reqs := []netmap.Request{
		{Role: node.RoleBP}, {Role: node.RoleBP}, {Role: node.RoleBP},
		{Role: node.RolePN}, {Role: node.RoleEN},
	}

	first, err := netmap.Assign(pool, reqs)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	second, err := netmap.Assign(pool, reqs)
	if err != nil {
		t.Fatalf("Assign (again): %v", err)
	}

	a, b := first.Placements(), second.Placements()
	if len(a) != len(b) {
		t.Fatalf("placement count drifted: %d then %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("placement %d drifted:\n first  %+v\n second %+v", i, a[i], b[i])
		}
	}
}
