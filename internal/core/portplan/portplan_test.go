package portplan

import (
	"strings"
	"testing"
)

func TestPlan(t *testing.T) {
	// golden bands: p2p 30010,30020,.. (etcd=+1) ; rpc 40010,40020,.. (ws+1,auth+2)
	p, err := Plan(1, 30010, 10, 40010, 10, DefaultReservation)
	if err != nil {
		t.Fatal(err)
	}
	if p.P2P != 30010 || p.Etcd != 30011 || p.HTTP != 40010 || p.WS != 40011 || p.Auth != 40012 {
		t.Errorf("node1 ports: %+v", p)
	}
	p2, _ := Plan(2, 30010, 10, 40010, 10, DefaultReservation)
	if p2.P2P != 30020 || p2.Etcd != 30021 || p2.HTTP != 40020 {
		t.Errorf("node2 ports: %+v", p2)
	}
}

func TestPlan_Rejects(t *testing.T) {
	if _, err := Plan(0, 30010, 10, 40010, 10, DefaultReservation); err == nil {
		t.Error("index 0 should error")
	}
	if _, err := Plan(1, 30010, 1, 40010, 10, DefaultReservation); err == nil {
		t.Error("p2p_step 1 leaves no room for etcd; should error")
	}
	// A family whose nodes embed etcd needs three consecutive ports (p2p, the
	// etcd peer at +1, the client at +2). A step of two passed the old global
	// rule and put the next node's p2p on this node's etcd client.
	poa := Reservation{P2PSpan: 3, RPCSpan: 3}
	if _, err := Plan(1, 30010, 2, 40010, 10, poa); err == nil {
		t.Error("p2p_step 2 is one short for an etcd family; should error")
	}
	if p, err := Plan(1, 30010, 10, 40010, 10, poa); err != nil {
		t.Errorf("etcd family with room: %v", err)
	} else if p.Etcd != 30011 || p.EtcdClient != 30012 {
		t.Errorf("etcd ports = peer %d client %d, want 30011/30012", p.Etcd, p.EtcdClient)
	}
	// A family that does not embed etcd reserves nothing for it.
	wbftOnly := Reservation{P2PSpan: 1, RPCSpan: 3}
	if p, err := Plan(1, 30010, 1, 40010, 10, wbftOnly); err != nil {
		t.Errorf("a one-port family should accept step 1: %v", err)
	} else if p.Etcd != 0 || p.EtcdClient != 0 {
		t.Errorf("no etcd ports expected, got peer %d client %d", p.Etcd, p.EtcdClient)
	}
	if _, err := Plan(1, 30010, 10, 40010, 2, DefaultReservation); err == nil {
		t.Error("rpc_step 2 has no room for auth; should error")
	}
}

func TestValidate(t *testing.T) {
	// golden: disjoint bands, step 10 -> no collision across 5 nodes.
	var golden []Ports
	for i := 1; i <= 5; i++ {
		p, _ := Plan(i, 30010, 10, 40010, 10, DefaultReservation)
		golden = append(golden, p)
	}
	if err := Validate(golden); err != nil {
		t.Errorf("golden plan should be collision-free: %v", err)
	}

	// etcd (p2p+1) colliding with the next node's p2p when packed one apart.
	packed := []Ports{{P2P: 100, Etcd: 101, HTTP: 200, WS: 201, Auth: 202}, {P2P: 101, Etcd: 102, HTTP: 210, WS: 211, Auth: 212}}
	if err := Validate(packed); err == nil || !strings.Contains(err.Error(), "101") {
		t.Errorf("etcd/p2p collision should error, got %v", err)
	}

	// p2p band overlapping the RPC band (etcd hits http) — the real bug hit.
	overlap := []Ports{{P2P: 51500, Etcd: 51501, HTTP: 51502, WS: 51503, Auth: 51504}, {P2P: 51501, Etcd: 51502, HTTP: 51600, WS: 51601, Auth: 51602}}
	if err := Validate(overlap); err == nil {
		t.Error("overlapping p2p/rpc bands should error")
	}
}
