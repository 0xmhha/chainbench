package node_test

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
	"strings"
	"testing"
)

// tiered builds bp1 bp2 pn1 en1 en2 on one host.
func tiered(t *testing.T) *node.Map {
	t.Helper()
	pool := resource.Pool{
		Hosts: []resource.Host{{Addr: "127.0.0.1"}},
		Slots: 8,
		Ports: resource.Bands{P2P: resource.Band{Base: 31000, Step: 10}, RPC: resource.Band{Base: 8600, Step: 10}},
	}
	m, err := resource.Assign(pool, []resource.Request{
		{Role: node.RoleBP}, {Role: node.RoleBP},
		{Role: node.RolePN},
		{Role: node.RoleEN}, {Role: node.RoleEN},
	})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	return m
}

func labels(t *testing.T, m *node.Map, p node.Peering, of node.Label) []string {
	t.Helper()
	peers, err := p.Peers(m, of)
	if err != nil {
		t.Fatalf("Peers(%s): %v", of, err)
	}
	out := make([]string, 0, len(peers))
	for _, l := range peers {
		out = append(out, string(l))
	}
	return out
}

// TestPeering_MeshListsTheWholeNetwork pins the entry that looks like a bug and
// is not: the list a node carries includes the node itself, because that is
// what every composition has written and the client ignores its own entry.
// Dropping it would change the launch arguments of every existing network.
func TestPeering_MeshListsTheWholeNetwork(t *testing.T) {
	m := tiered(t)
	got := labels(t, m, node.Mesh, "node1")
	want := []string{"node1", "node2", "node3", "node4", "node5"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("mesh peers = %v, want %v", got, want)
	}
	// Every node gets the same list, in the same order.
	for _, l := range []node.Label{"node2", "node5"} {
		if other := labels(t, m, node.Mesh, l); strings.Join(other, ",") != strings.Join(want, ",") {
			t.Fatalf("%s mesh peers = %v, want the same list", l, other)
		}
	}
}

// TestPeering_ProxiedKeepsEndpointsAwayFromProducers is the property the tier
// exists for: a transaction reaches the chain through en and travels inward, so
// an exposed endpoint must not be a route to a validator.
func TestPeering_ProxiedKeepsEndpointsAwayFromProducers(t *testing.T) {
	m := tiered(t)

	// node1/node2 are bp, node3 is pn, node4/node5 are en.
	//
	// A producer keeps its peers among the other producers plus the tier. It
	// was measured: with the pn as a bp's only peer, consensus never forms —
	// a pn is not a validator and does not carry consensus traffic.
	if got := labels(t, m, node.Proxied, "node1"); strings.Join(got, ",") != "node2,node3" {
		t.Fatalf("bp peers = %v, want the other producer and the pn", got)
	}
	if got := labels(t, m, node.Proxied, "node4"); strings.Join(got, ",") != "node3" {
		t.Fatalf("en peers = %v, want only the pn", got)
	}
	// The tier is the only role that sees both sides.
	if got := labels(t, m, node.Proxied, "node3"); strings.Join(got, ",") != "node1,node2,node4,node5" {
		t.Fatalf("pn peers = %v, want both tiers", got)
	}
	// Stated as the invariant rather than as positions: no en lists any bp.
	for _, en := range []node.Label{"node4", "node5"} {
		for _, peer := range labels(t, m, node.Proxied, en) {
			pl, _ := m.Lookup(node.Label(peer))
			if pl.Role == node.RoleBP {
				t.Fatalf("%s lists producer %s", en, peer)
			}
		}
	}
}

func TestPeering_ValidateRejectsWhatCannotRun(t *testing.T) {
	m := tiered(t)

	// A family without a proxy tier: poa puts etcd in that place, so a pn is a
	// declaration that will not be honoured.
	noPN := func(r node.Role) bool { return r != node.RolePN }
	err := node.Mesh.Validate(m, noPN)
	if err == nil || !strings.Contains(err.Error(), "pn") {
		t.Fatalf("want a rejection naming pn, got %v", err)
	}

	// Proxied with nothing in the middle is a tier that does not exist.
	pool := resource.Pool{
		Hosts: []resource.Host{{Addr: "127.0.0.1"}}, Slots: 4,
		Ports: resource.Bands{P2P: resource.Band{Base: 31000, Step: 10}, RPC: resource.Band{Base: 8600, Step: 10}},
	}
	flat, err := resource.Assign(pool, []resource.Request{{Role: node.RoleBP}, {Role: node.RoleEN}})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if err := node.Proxied.Validate(flat, nil); err == nil {
		t.Fatal("proxied without a pn must be refused, not demoted to mesh")
	}
	// The same network is fine meshed.
	if err := node.Mesh.Validate(flat, nil); err != nil {
		t.Fatalf("mesh on a two-tier network: %v", err)
	}

	// The mirror of the proxied-without-pn rejection: a pn declared under mesh
	// is a tier that will not take effect, so mesh must refuse it rather than
	// silently let endpoints dial producers (WA9). m carries a pn.
	if err := node.Mesh.Validate(m, nil); err == nil || !strings.Contains(err.Error(), "mesh") {
		t.Fatalf("mesh with a pn must be refused, got %v", err)
	}
}

func TestParsePeering(t *testing.T) {
	for in, want := range map[string]node.Peering{"": node.Mesh, "mesh": node.Mesh, "proxied": node.Proxied} {
		got, err := node.ParsePeering(in)
		if err != nil || got != want {
			t.Fatalf("ParsePeering(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := node.ParsePeering("star"); err == nil {
		t.Fatal("an unknown peering must not fall back to a default")
	}
}

func TestStaticNodes_FormatterOwnsTheKeyMaterial(t *testing.T) {
	m := tiered(t)
	// The formatter stands in for the caller that holds both the map and the
	// keyring; a peer it cannot express is skipped, as the assemblies this
	// replaces did.
	enode := func(p node.Placement) (string, bool) {
		if p.Label == "node2" {
			return "", false // no key material yet
		}
		return string(p.Label) + "@" + p.Host + ":" + itoa(p.Ports.P2P), true
	}
	got, err := node.Proxied.StaticNodes(m, "node3", enode)
	if err != nil {
		t.Fatalf("StaticNodes: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("static nodes = %v, want the three expressible peers", got)
	}
	for _, e := range got {
		if strings.HasPrefix(e, "node2@") {
			t.Fatalf("a peer with no key must be skipped, got %v", got)
		}
	}
	if _, err := node.Mesh.StaticNodes(m, "node1", nil); err == nil {
		t.Fatal("assembling without a formatter must error rather than return an empty list")
	}
	if _, err := node.Mesh.StaticNodes(m, "node9", enode); err == nil {
		t.Fatal("a node outside the network must error")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for ; n > 0; n /= 10 {
		b = append([]byte{byte('0' + n%10)}, b...)
	}
	return string(b)
}

// bridged builds nine producers on one host.
func bridged(t *testing.T) *node.Map {
	t.Helper()
	pool := resource.Pool{
		Hosts: []resource.Host{{Addr: "127.0.0.1"}},
		Slots: 12,
		Ports: resource.Bands{P2P: resource.Band{Base: 31000, Step: 10}, RPC: resource.Band{Base: 8600, Step: 10}},
	}
	reqs := make([]resource.Request, 9)
	for i := range reqs {
		reqs[i] = resource.Request{Role: node.RoleBP}
	}
	m, err := resource.Assign(pool, reqs)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	return m
}

// TestPeering_GroupsWireOnlyWhatIsDeclared: two groups of four joined by the
// node in both — each side lists only its own members and the bridge, the
// bridge lists everyone, and the declaration survives its own string form.
func TestPeering_GroupsWireOnlyWhatIsDeclared(t *testing.T) {
	m := bridged(t)
	p, err := node.GroupsPeering([][]string{
		{"node1", "node2", "node3", "node4", "node5"},
		{"node5", "node6", "node7", "node8", "node9"},
	})
	if err != nil {
		t.Fatalf("GroupsPeering: %v", err)
	}
	if err := p.Validate(m, nil); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	for of, want := range map[node.Label]string{
		"node1": "node2,node3,node4,node5",
		"node5": "node1,node2,node3,node4,node6,node7,node8,node9",
		"node9": "node5,node6,node7,node8",
	} {
		if got := strings.Join(labels(t, m, p, of), ","); got != want {
			t.Errorf("%s peers = %s, want %s", of, got, want)
		}
		if n, ok := p.DeclaredPeers(of); !ok || n != len(strings.Split(want, ",")) {
			t.Errorf("%s DeclaredPeers = %d, %v", of, n, ok)
		}
	}
	back, err := node.ParsePeering(string(p))
	if err != nil || back != p {
		t.Fatalf("ParsePeering(%q) = %q, %v", p, back, err)
	}
	if _, ok := node.Mesh.DeclaredPeers("node1"); ok {
		t.Error("mesh claims a declared peer count")
	}
}

// TestPeering_GroupsRefuseWhatCannotRun: a declaration naming a node the
// network lacks, leaving one out, or splitting the producers from the start is
// refused before anything is written.
func TestPeering_GroupsRefuseWhatCannotRun(t *testing.T) {
	m := bridged(t)
	for name, c := range map[string]struct {
		groups [][]string
		want   string
	}{
		"unknown node":     {[][]string{{"node1", "node2", "node3", "node4", "node5", "node6", "node7", "node8", "node9", "node10"}}, "not in this network"},
		"node left out":    {[][]string{{"node1", "node2", "node3", "node4", "node5", "node6", "node7", "node8"}}, "leave node9 out"},
		"split from start": {[][]string{{"node1", "node2", "node3", "node4"}, {"node5", "node6", "node7", "node8", "node9"}}, "unreachable"},
	} {
		p, err := node.GroupsPeering(c.groups)
		if err != nil {
			t.Fatalf("%s: GroupsPeering: %v", name, err)
		}
		if err := p.Validate(m, nil); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: Validate = %v, want %q", name, err, c.want)
		}
	}
	for name, g := range map[string][][]string{
		"lone node":  {{"node1"}},
		"role alias": {{"bp1", "bp2"}},
		"duplicate":  {{"node1", "node1"}},
		"no groups":  nil,
	} {
		if _, err := node.GroupsPeering(g); err == nil {
			t.Errorf("%s: GroupsPeering accepted %v", name, g)
		}
	}
}
