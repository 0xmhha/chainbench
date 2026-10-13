package resource_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestInventoryPinnedServersReserveBeforeAutomaticRequests(t *testing.T) {
	inv := threeByTwo(t)
	m, err := inv.Assign([]resource.Request{{Role: node.RoleBP}, {Role: node.RoleBP, Server: "a"}, {Role: node.RoleBP, Server: "a"}, {Role: node.RoleEN}}, "pinned")
	if err != nil {
		t.Fatal(err)
	}
	placements := m.Placements()
	for i, want := range []struct {
		host string
		p2p  int
	}{{"10.0.0.2", 31000}, {"10.0.0.1", 31000}, {"10.0.0.1", 31010}, {"10.0.0.3", 31000}} {
		if placements[i].Host != want.host || placements[i].Ports.P2P != want.p2p {
			t.Errorf("node%d placement=%+v, want %+v", i+1, placements[i], want)
		}
	}
	if inv.Usage().Used != 4 {
		t.Fatal(inv.Usage())
	}
}

func TestInventoryPinnedFailureTakesNothing(t *testing.T) {
	for _, sample := range []struct {
		name string
		reqs []resource.Request
		full bool
	}{
		{"unknown", []resource.Request{{Role: node.RoleBP, Server: "missing"}}, false},
		{"host-full", []resource.Request{{Role: node.RoleBP, Server: "a"}}, true},
		{"role-invalid", []resource.Request{{Role: node.RoleBP, Server: "b"}, {Role: "unsupported", Server: "c"}}, false},
	} {
		t.Run(sample.name, func(t *testing.T) {
			inv := threeByTwo(t)
			if sample.full {
				inv.Adopt([]resource.Allocation{{Network: "holder", Host: "a", P2P: 31000}, {Network: "holder", Host: "a", P2P: 31010}})
			}
			before := inv.Usage().Used
			_, err := inv.Assign(sample.reqs, "rejected")
			if err == nil {
				t.Fatal("invalid placement accepted")
			}
			if sample.full && !errors.Is(err, resource.ErrFull) {
				t.Fatalf("want full-host refusal: %v", err)
			}
			if inv.Usage().Used != before || inv.Usage().ByNetwork["rejected"] != 0 {
				t.Fatalf("failed assignment leaked claims: %+v", inv.Usage())
			}
		})
	}
}

func TestPoolRejectsAmbiguousServerIdentity(t *testing.T) {
	for _, hosts := range [][]resource.Host{
		{{Name: "a", Addr: "10.0.0.1"}, {Name: "a", Addr: "10.0.0.2"}},
		{{Name: "a", Addr: "10.0.0.1"}, {Name: "b", Addr: "10.0.0.1"}},
	} {
		p := resource.Pool{Hosts: hosts, Slots: 1, Ports: resource.Bands{P2P: resource.Band{Base: 31000, Step: 10}, RPC: resource.Band{Base: 8600, Step: 10}}}
		_, err := resource.NewInventory(p)
		if err == nil || !strings.Contains(err.Error(), "duplicate") {
			t.Fatalf("ambiguous pool accepted: %v", err)
		}
	}
}
