package resource_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

// threeByTwo is a set of three hosts with two slots each: six nodes.
func threeByTwo(t *testing.T) *resource.Inventory {
	t.Helper()
	inv, err := resource.NewInventory(resource.Pool{
		Hosts: []resource.Host{{Name: "a", Addr: "10.0.0.1"}, {Name: "b", Addr: "10.0.0.2"}, {Name: "c", Addr: "10.0.0.3"}},
		Slots: 2,
		Ports: resource.Bands{P2P: resource.Band{Base: 31000, Step: 10}, RPC: resource.Band{Base: 8600, Step: 10}},
	})
	if err != nil {
		t.Fatalf("NewInventory: %v", err)
	}
	return inv
}

// TestInventory_TakeConsumesHostsBeforeSlots pins the order: a take and an
// Assign must agree on where node i lands, or the plan lies about the take.
func TestInventory_TakeConsumesHostsBeforeSlots(t *testing.T) {
	inv := threeByTwo(t)
	got, err := inv.Take(4, "net-a")
	if err != nil {
		t.Fatalf("Take: %v", err)
	}
	want := []resource.Slot{{Host: "a", Index: 1}, {Host: "b", Index: 1}, {Host: "c", Index: 1}, {Host: "a", Index: 2}}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("slot %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	u := inv.Usage()
	if u.Cap != 6 || u.Used != 4 || u.Free != 2 || u.ByNetwork["net-a"] != 4 {
		t.Errorf("usage = %+v", u)
	}
}

// TestInventory_FullNamesTheHolders: "15 of 15" tells an operator nothing;
// the refusal must say which network to remove.
func TestInventory_FullNamesTheHolders(t *testing.T) {
	inv := threeByTwo(t)
	if _, err := inv.Take(4, "net-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := inv.Take(2, "net-b"); err != nil {
		t.Fatal(err)
	}
	if !inv.Usage().Full() {
		t.Fatal("six taken of six should be full")
	}
	_, err := inv.Take(1, "net-c")
	if !errors.Is(err, resource.ErrFull) {
		t.Fatalf("want ErrFull, got %v", err)
	}
	for _, want := range []string{"net-a holds 4", "net-b holds 2", "0 free"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal should say %q:\n%v", want, err)
		}
	}
	if inv.Usage().ByNetwork["net-c"] != 0 {
		t.Error("a refused take must take nothing")
	}
}

// TestInventory_ReleaseIsRemovalNotStop: stopping a node changes its pid,
// not its claim. Only removing the network gives the slots back.
func TestInventory_ReleaseIsRemovalNotStop(t *testing.T) {
	inv := threeByTwo(t)
	if _, err := inv.Take(3, "net-a"); err != nil {
		t.Fatal(err)
	}
	if n := inv.Release("net-a"); n != 3 {
		t.Fatalf("released %d, want 3", n)
	}
	if u := inv.Usage(); u.Used != 0 || u.Free != 6 {
		t.Errorf("after release usage = %+v", u)
	}
	if n := inv.Release("net-a"); n != 0 {
		t.Errorf("releasing twice returned %d", n)
	}
}

// TestInventory_AdoptDerivesFromRecords: a new process learns what is taken
// from what workspaces recorded — by host and p2p port — and never invents a
// claim of its own. A record from another set, or from a port outside the
// band, is not this inventory's business.
func TestInventory_AdoptDerivesFromRecords(t *testing.T) {
	inv := threeByTwo(t)
	inv.Adopt([]resource.Allocation{
		{Network: "old", Node: "node1", Host: "10.0.0.1", P2P: 31000}, // a, slot 1
		{Network: "old", Node: "node2", Host: "b", P2P: 31010},        // b by name, slot 2
		{Network: "other-set", Node: "node1", Host: "192.168.9.9", P2P: 31000},
		{Network: "off-band", Node: "node1", Host: "10.0.0.3", P2P: 31005},
		{Network: "too-far", Node: "node1", Host: "10.0.0.3", P2P: 31020}, // slot 3 of 2
	})
	u := inv.Usage()
	if u.Used != 2 || u.ByNetwork["old"] != 2 {
		t.Fatalf("usage = %+v, want exactly the two in-set records", u)
	}
	// The next take must skip what was adopted.
	got, err := inv.Take(1, "new")
	if err != nil {
		t.Fatal(err)
	}
	if got[0] != (resource.Slot{Host: "b", Index: 1}) {
		t.Errorf("first free slot = %+v, want b/1 (a/1 is adopted)", got[0])
	}
}

// TestInventory_AdoptKeepsTheFirstHolder: two workspaces claiming one slot is
// a real conflict on disk; the inventory reports the older claim rather than
// letting the newer one silently win.
func TestInventory_AdoptKeepsTheFirstHolder(t *testing.T) {
	inv := threeByTwo(t)
	inv.Adopt([]resource.Allocation{{Network: "first", Node: "node1", Host: "a", P2P: 31000}})
	inv.Adopt([]resource.Allocation{{Network: "second", Node: "node1", Host: "a", P2P: 31000}})
	u := inv.Usage()
	if u.ByNetwork["first"] != 1 || u.ByNetwork["second"] != 0 {
		t.Errorf("usage = %+v", u)
	}
}
