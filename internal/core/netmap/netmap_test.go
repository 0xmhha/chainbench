package netmap_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
)

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
		label    node.Label
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
		// The etcd port travels with the node; before this package the port
		// representations dropped it on the way to runtime.
		if p.Ports.Etcd != p.Ports.P2P+1 {
			t.Fatalf("%s etcd = %d, want %d", c.label, p.Ports.Etcd, p.Ports.P2P+1)
		}
	}
}

// TestAssign_OverCapacityNamesTheShortfall: wrapping onto ports already handed
// out produces a network where two nodes cannot both bind, found much later.

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
	want := map[node.Label]node.Label{
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
