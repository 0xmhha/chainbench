package place_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/place"
)

func validators(n int) []place.NodeReq {
	reqs := make([]place.NodeReq, n)
	for i := range reqs {
		reqs[i] = place.NodeReq{Name: "bp", Role: node.RoleValidator, Binary: "go-wbft"}
	}
	return reqs
}

func testConfig() place.Config {
	return place.Config{P2PBase: 30300, P2PStep: 2, RPCBase: 40300, RPCStep: 3}
}

func TestCapacity_MinValidators(t *testing.T) {
	a := place.New(testConfig())
	_, err := a.Allocate(validators(3), place.LocalStepped, place.Capacity{MinValidators: 4})
	if err == nil {
		t.Fatal("3 validators with MinValidators=4 must fail")
	}
}

func TestCapacity_MaxExceededRemote(t *testing.T) {
	cfg := testConfig()
	cfg.Hosts = []string{"10.0.0.1", "10.0.0.2"}
	a := place.New(cfg)
	_, err := a.Allocate(validators(3), place.RemotePerHost, place.Capacity{MinValidators: 1})
	if err == nil {
		t.Fatal("3 nodes on 2 hosts must fail")
	}
}

func TestLocalStepped(t *testing.T) {
	a := place.New(testConfig())
	got, err := a.Allocate(validators(4), place.LocalStepped, place.Capacity{MinValidators: 4})
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("got %d placements, want 4", len(got))
	}
	// Node 1 = base; node 2 = base+step; etcd = p2p+1.
	if got[0].Ports.P2P != 30300 || got[0].Ports.Etcd != 30301 {
		t.Fatalf("node1 ports = %+v", got[0].Ports)
	}
	if got[1].Ports.P2P != 30302 {
		t.Fatalf("node2 p2p = %d, want 30302", got[1].Ports.P2P)
	}
	if got[0].Host != "127.0.0.1" {
		t.Fatalf("host = %q, want 127.0.0.1", got[0].Host)
	}
	// No collisions across the set.
	seen := map[int]bool{}
	for _, p := range got {
		for _, port := range []int{p.Ports.P2P, p.Ports.Etcd, p.Ports.HTTP, p.Ports.WS, p.Ports.Auth} {
			if seen[port] {
				t.Fatalf("port %d collides", port)
			}
			seen[port] = true
		}
	}
}

func TestRemotePerHost(t *testing.T) {
	cfg := testConfig()
	cfg.Hosts = []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
	a := place.New(cfg)
	got, err := a.Allocate(validators(3), place.RemotePerHost, place.Capacity{MinValidators: 1})
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	for i, p := range got {
		if p.Host != cfg.Hosts[i] {
			t.Fatalf("node %d host = %q, want %q", i, p.Host, cfg.Hosts[i])
		}
		// Same port on every host.
		if p.Ports.P2P != got[0].Ports.P2P {
			t.Fatalf("remote nodes must share ports: %d vs %d", p.Ports.P2P, got[0].Ports.P2P)
		}
	}
}

func TestLocalOSAssigned(t *testing.T) {
	a := place.New(testConfig())
	got, err := a.Allocate(validators(4), place.LocalOSAssigned, place.Capacity{MinValidators: 4})
	if err != nil {
		t.Fatalf("Allocate: %v", err)
	}
	seen := map[int]bool{}
	for _, p := range got {
		if p.Host != "127.0.0.1" {
			t.Fatalf("host = %q", p.Host)
		}
		for _, port := range []int{p.Ports.P2P, p.Ports.HTTP} {
			if port == 0 {
				t.Fatal("OS-assigned port must be non-zero")
			}
			if seen[port] {
				t.Fatalf("OS-assigned port %d collides", port)
			}
			seen[port] = true
		}
	}
}
