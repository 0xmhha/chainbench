package netmap_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// The allocator netmap replaced had two deterministic modes: a local network on
// the loopback with stepped ports, and a server set with one node per host on
// identical ports. They are this grid read two ways, and these are the values
// it produced — kept as goldens now that the allocator itself is gone, because
// the numbers are what an existing network is already running on.
//
// If these change, every composed network moves its ports, which is a decision
// and not a refactor.
func TestAssign_ReproducesTheLocalSteppedGolden(t *testing.T) {
	m, err := netmap.Assign(netmap.Pool{
		Hosts: []netmap.Host{{Name: "local", Addr: "127.0.0.1"}},
		Slots: 6,
		Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10},
	}, requests(6))
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	for i, p := range m.Placements() {
		wantP2P, wantHTTP := 31000+i*10, 8600+i*10
		if p.Host != "127.0.0.1" || p.Ports.P2P != wantP2P || p.Ports.HTTP != wantHTTP {
			t.Fatalf("%s = %s p2p=%d http=%d, want 127.0.0.1 p2p=%d http=%d",
				p.Label, p.Host, p.Ports.P2P, p.Ports.HTTP, wantP2P, wantHTTP)
		}
		if p.Ports.Etcd != wantP2P+1 {
			t.Fatalf("%s etcd = %d, want %d", p.Label, p.Ports.Etcd, wantP2P+1)
		}
	}
}

func TestAssign_ReproducesTheSetGolden(t *testing.T) {
	hosts := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"}
	pool := netmap.Pool{Slots: 1, Ports: netmap.Bands{P2PBase: 31000, P2PStep: 10, RPCBase: 8600, RPCStep: 10}}
	for _, h := range hosts {
		pool.Hosts = append(pool.Hosts, netmap.Host{Addr: h})
	}
	m, err := netmap.Assign(pool, requests(len(hosts)))
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}
	// One node per host, every node on the same ports — the server set shape.
	for i, p := range m.Placements() {
		if p.Host != hosts[i] {
			t.Fatalf("%s host = %s, want %s", p.Label, p.Host, hosts[i])
		}
		if p.Ports.P2P != 31000 || p.Ports.HTTP != 8600 {
			t.Fatalf("%s ports = %+v, want the first slot for every host", p.Label, p.Ports)
		}
	}
}

func requests(n int) []netmap.Request {
	out := make([]netmap.Request, n)
	for i := range out {
		out[i] = netmap.Request{Role: node.RoleValidator}
	}
	return out
}
