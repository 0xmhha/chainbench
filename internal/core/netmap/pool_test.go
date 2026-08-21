package netmap_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestAssign_WalksHostsFirst is the design's worked example (§2.2b) as a
// table: 5 hosts × 4 bases, 15 nodes. The walk exhausts the host list before
// advancing the base, so node6 lands back on host1 at the next base.
func TestAssign_WalksHostsFirst(t *testing.T) {
	pool := netmap.Pool{
		Hosts:     []string{"ip1", "ip2", "ip3", "ip4", "ip5"},
		PortBases: []int{8080, 8180, 8280, 8380},
		DataRoot:  "/data/chainbench",
	}
	roles := make([]node.Role, 15)
	for i := range roles {
		roles[i] = node.RoleBP
	}

	m, err := netmap.Assign(pool, roles)
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}

	want := []struct {
		label string
		host  string
		p2p   int
	}{
		{"node1", "ip1", 8080}, {"node2", "ip2", 8080}, {"node5", "ip5", 8080},
		{"node6", "ip1", 8180}, // hosts exhausted: same host as node1, next base
		{"node10", "ip5", 8180},
		{"node11", "ip1", 8280}, {"node15", "ip5", 8280},
	}
	for _, w := range want {
		p, ok := m.Lookup(netmap.NodeLabel(w.label))
		if !ok {
			t.Fatalf("%s was not placed", w.label)
		}
		if p.Host != w.host || p.Ports.P2P != w.p2p {
			t.Errorf("%s = (%s, %d), want (%s, %d)", w.label, p.Host, p.Ports.P2P, w.host, w.p2p)
		}
		if p.DataDir != "/data/chainbench/"+w.label {
			t.Errorf("%s dataDir = %s", w.label, p.DataDir)
		}
	}

	// node1 and node6 share a host; their full port sets must not touch.
	n1, _ := m.Lookup("node1")
	n6, _ := m.Lookup("node6")
	if n1.Host != n6.Host {
		t.Fatal("the example's premise broke: node1 and node6 should share a host")
	}
	if n1.Ports.Metrics >= n6.Ports.P2P {
		t.Errorf("port sets overlap on a shared host: node1 ends at %d, node6 starts at %d",
			n1.Ports.Metrics, n6.Ports.P2P)
	}
}

// TestAssign_IsDeterministic: same pool, same roles, same map — a re-run must
// not produce a different layout.
func TestAssign_IsDeterministic(t *testing.T) {
	pool := netmap.Pool{Hosts: []string{"a", "b"}, PortBases: []int{9000, 9010}}
	roles := []node.Role{node.RoleBP, node.RoleBP, node.RoleEN}

	first, err := netmap.Assign(pool, roles)
	if err != nil {
		t.Fatal(err)
	}
	second, err := netmap.Assign(pool, roles)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range first.Labels() {
		a, _ := first.Lookup(l)
		b, _ := second.Lookup(l)
		if a != b {
			t.Errorf("%s moved between runs: %+v vs %+v", l, a, b)
		}
	}
}

// TestAssign_OverflowIsACountedRefusal: asking for more nodes than the pool
// holds must say how many are missing, and place nothing.
func TestAssign_OverflowIsACountedRefusal(t *testing.T) {
	pool := netmap.Pool{Hosts: []string{"a", "b"}, PortBases: []int{9000}}
	roles := make([]node.Role, 5) // capacity is 2
	for i := range roles {
		roles[i] = node.RoleBP
	}
	_, err := netmap.Assign(pool, roles)
	if err == nil {
		t.Fatal("an overflowing assignment was accepted")
	}
	for _, want := range []string{"5", "capacity of 2", "3 short"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("overflow error should mention %q: %v", want, err)
		}
	}
}

// TestPool_Validate rejects the pools that cannot assign safely.
func TestPool_Validate(t *testing.T) {
	cases := []struct {
		name string
		pool netmap.Pool
		want string
	}{
		{"no hosts", netmap.Pool{PortBases: []int{9000}}, "no hosts"},
		{"no bases", netmap.Pool{Hosts: []string{"a"}}, "no port bases"},
		{"duplicate host", netmap.Pool{Hosts: []string{"a", "a"}, PortBases: []int{9000}}, "twice"},
		{
			// Bases 8080 and 8081 collide the moment two nodes share a host:
			// node A's etcd (8081) is node B's p2p. The user-level list is of
			// bases, not single ports, and spacing is what makes that safe.
			"bases too close",
			netmap.Pool{Hosts: []string{"a"}, PortBases: []int{8080, 8081}},
			"spans",
		},
		{"bad base", netmap.Pool{Hosts: []string{"a"}, PortBases: []int{0}}, "not a valid port"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.pool.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Validate = %v, want mention of %q", err, tc.want)
			}
		})
	}
}
