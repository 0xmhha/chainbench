package netmap_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// tiered builds bp1 bp2 pn1 en1 en2 on one host.
func tiered(t *testing.T) *netmap.Map {
	t.Helper()
	pool := netmap.Pool{
		Hosts: []netmap.Host{{Addr: "127.0.0.1"}},
		Slots: 8,
		Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10},
	}
	m, err := netmap.Assign(pool, []netmap.Request{
		{Role: node.RoleBP}, {Role: node.RoleBP},
		{Role: node.RolePN},
		{Role: node.RoleEN}, {Role: node.RoleEN},
	})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	return m
}

func labels(t *testing.T, m *netmap.Map, p netmap.Peering, of netmap.NodeLabel) []string {
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
	got := labels(t, m, netmap.Mesh, "node1")
	want := []string{"node1", "node2", "node3", "node4", "node5"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("mesh peers = %v, want %v", got, want)
	}
	// Every node gets the same list, in the same order.
	for _, l := range []netmap.NodeLabel{"node2", "node5"} {
		if other := labels(t, m, netmap.Mesh, l); strings.Join(other, ",") != strings.Join(want, ",") {
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
	if got := labels(t, m, netmap.Proxied, "node1"); strings.Join(got, ",") != "node3" {
		t.Fatalf("bp peers = %v, want only the pn", got)
	}
	if got := labels(t, m, netmap.Proxied, "node4"); strings.Join(got, ",") != "node3" {
		t.Fatalf("en peers = %v, want only the pn", got)
	}
	// The tier is the only role that sees both sides.
	if got := labels(t, m, netmap.Proxied, "node3"); strings.Join(got, ",") != "node1,node2,node4,node5" {
		t.Fatalf("pn peers = %v, want both tiers", got)
	}
	// Stated as the invariant rather than as positions: no en lists any bp.
	for _, en := range []netmap.NodeLabel{"node4", "node5"} {
		for _, peer := range labels(t, m, netmap.Proxied, en) {
			pl, _ := m.Lookup(netmap.NodeLabel(peer))
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
	err := netmap.Mesh.Validate(m, noPN)
	if err == nil || !strings.Contains(err.Error(), "pn") {
		t.Fatalf("want a rejection naming pn, got %v", err)
	}

	// Proxied with nothing in the middle is a tier that does not exist.
	pool := netmap.Pool{
		Hosts: []netmap.Host{{Addr: "127.0.0.1"}}, Slots: 4,
		Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10},
	}
	flat, err := netmap.Assign(pool, []netmap.Request{{Role: node.RoleBP}, {Role: node.RoleEN}})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	if err := netmap.Proxied.Validate(flat, nil); err == nil {
		t.Fatal("proxied without a pn must be refused, not demoted to mesh")
	}
	// The same network is fine meshed.
	if err := netmap.Mesh.Validate(flat, nil); err != nil {
		t.Fatalf("mesh on a two-tier network: %v", err)
	}
}

func TestParsePeering(t *testing.T) {
	for in, want := range map[string]netmap.Peering{"": netmap.Mesh, "mesh": netmap.Mesh, "proxied": netmap.Proxied} {
		got, err := netmap.ParsePeering(in)
		if err != nil || got != want {
			t.Fatalf("ParsePeering(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := netmap.ParsePeering("star"); err == nil {
		t.Fatal("an unknown peering must not fall back to a default")
	}
}

func TestStaticNodes_FormatterOwnsTheKeyMaterial(t *testing.T) {
	m := tiered(t)
	// The formatter stands in for the caller that holds both the map and the
	// keyring; a peer it cannot express is skipped, as the assemblies this
	// replaces did.
	enode := func(p netmap.Placement) (string, bool) {
		if p.Label == "node2" {
			return "", false // no key material yet
		}
		return string(p.Label) + "@" + p.Host + ":" + itoa(p.Ports.P2P), true
	}
	got, err := netmap.Proxied.StaticNodes(m, "node3", enode)
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
	if _, err := netmap.Mesh.StaticNodes(m, "node1", nil); err == nil {
		t.Fatal("assembling without a formatter must error rather than return an empty list")
	}
	if _, err := netmap.Mesh.StaticNodes(m, "node9", enode); err == nil {
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
