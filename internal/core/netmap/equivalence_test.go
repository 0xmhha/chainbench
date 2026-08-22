package netmap_test

import (
	"fmt"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/place"
)

// The two deterministic place modes are this grid read differently: a local
// network is one host with many slots, a fleet is many hosts with one slot. If
// Assign did not reproduce them byte for byte, NM3 would move every port on
// every existing network while claiming to be a refactor — so the equivalence
// is a test, not a comment. (place.LocalOSAssigned is not covered: it asks the
// OS for free ports, which is a different strategy rather than a grid.)
func TestAssign_ReproducesPlaceLocalStepped(t *testing.T) {
	const n = 6
	bands := netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10}

	old, err := place.New(place.Config{
		P2PBase: bands.P2PBase, P2PStep: bands.P2PStep,
		RPCBase: bands.RPCBase, RPCStep: bands.RPCStep,
	}).Allocate(placeReqs(n), place.LocalStepped, place.Capacity{})
	if err != nil {
		t.Fatalf("place.Allocate: %v", err)
	}

	m, err := netmap.Assign(netmap.Pool{
		Hosts: []netmap.Host{{Name: "local", Addr: "127.0.0.1"}},
		Slots: n,
		Ports: bands,
	}, netmapReqs(n))
	if err != nil {
		t.Fatalf("netmap.Assign: %v", err)
	}
	assertSamePlacement(t, old, m)
}

func TestAssign_ReproducesPlaceRemotePerHost(t *testing.T) {
	hosts := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"}
	bands := netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10}

	old, err := place.New(place.Config{
		P2PBase: bands.P2PBase, P2PStep: bands.P2PStep,
		RPCBase: bands.RPCBase, RPCStep: bands.RPCStep,
		Hosts: hosts,
	}).Allocate(placeReqs(len(hosts)), place.RemotePerHost, place.Capacity{})
	if err != nil {
		t.Fatalf("place.Allocate: %v", err)
	}

	pool := netmap.Pool{Slots: 1, Ports: bands}
	for _, h := range hosts {
		pool.Hosts = append(pool.Hosts, netmap.Host{Addr: h})
	}
	m, err := netmap.Assign(pool, netmapReqs(len(hosts)))
	if err != nil {
		t.Fatalf("netmap.Assign: %v", err)
	}
	assertSamePlacement(t, old, m)
}

func placeReqs(n int) []place.NodeReq {
	out := make([]place.NodeReq, n)
	for i := range out {
		out[i] = place.NodeReq{Name: fmt.Sprintf("node%d", i+1), Role: node.RoleValidator}
	}
	return out
}

func netmapReqs(n int) []netmap.Request {
	out := make([]netmap.Request, n)
	for i := range out {
		out[i] = netmap.Request{Role: node.RoleValidator}
	}
	return out
}

func assertSamePlacement(t *testing.T, old []place.NodePlacement, m *netmap.Map) {
	t.Helper()
	got := m.Placements()
	if len(got) != len(old) {
		t.Fatalf("assigned %d nodes, place assigned %d", len(got), len(old))
	}
	for i, want := range old {
		have := got[i]
		if string(have.Label) != want.Name {
			t.Fatalf("node %d label = %q, want %q", i+1, have.Label, want.Name)
		}
		if have.Host != want.Host {
			t.Fatalf("%s host = %q, want %q", have.Label, have.Host, want.Host)
		}
		if have.Ports != want.Ports {
			t.Fatalf("%s ports = %+v, want %+v", have.Label, have.Ports, want.Ports)
		}
	}
}
