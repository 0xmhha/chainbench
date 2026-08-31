package mcp

import (
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func sampleNet(name string) node.NodeSet {
	return node.NodeSet{
		Chain: "wbft", Network: name,
		Nodes: []node.Node{{
			Index: 1, Role: node.RoleEndpoint, Host: "10.0.0.1",
			RPCURL: "http://10.0.0.1:8545",
			Auth:   map[string]any{"type": "api-key", "env": "K"},
		}},
		Capabilities: []string{"rpc"},
	}
}

func TestNetworkRegistry_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := saveNetwork(dir, sampleNet("prod")); err != nil {
		t.Fatal(err)
	}
	got, err := loadNetwork(dir, "prod")
	if err != nil {
		t.Fatal(err)
	}
	if got.Network != "prod" || got.Chain != "wbft" || len(got.Nodes) != 1 {
		t.Fatalf("round-trip lost data: %+v", got)
	}
	// auth descriptor persists (env-var name only)
	if got.Nodes[0].Auth["env"] != "K" {
		t.Errorf("auth not persisted: %v", got.Nodes[0].Auth)
	}
}

func TestNetworkRegistry_ListAndRemove(t *testing.T) {
	dir := t.TempDir()
	// empty dir lists cleanly
	if ns, err := listNetworks(dir); err != nil || len(ns) != 0 {
		t.Fatalf("empty list: %v %v", ns, err)
	}
	for _, n := range []string{"beta", "alpha"} {
		if err := saveNetwork(dir, sampleNet(n)); err != nil {
			t.Fatal(err)
		}
	}
	list, err := listNetworks(dir)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %v %v", list, err)
	}
	if list[0].Network != "alpha" || list[1].Network != "beta" {
		t.Errorf("not sorted by name: %v", list)
	}
	if err := removeNetwork(dir, "alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := loadNetwork(dir, "alpha"); !errors.Is(err, ErrNetworkNotFound) {
		t.Errorf("removed network should be not-found: %v", err)
	}
}

func TestNetworkRegistry_Rejects(t *testing.T) {
	dir := t.TempDir()
	// reserved name
	if err := saveNetwork(dir, sampleNet("local")); !errors.Is(err, ErrReservedName) {
		t.Errorf("reserved name: %v", err)
	}
	// invalid name
	if err := saveNetwork(dir, sampleNet("Bad Name")); !errors.Is(err, ErrInvalidName) {
		t.Errorf("invalid name: %v", err)
	}
	// missing on load/remove
	if _, err := loadNetwork(dir, "nope"); !errors.Is(err, ErrNetworkNotFound) {
		t.Errorf("missing load: %v", err)
	}
	if err := removeNetwork(dir, "nope"); !errors.Is(err, ErrNetworkNotFound) {
		t.Errorf("missing remove: %v", err)
	}
	if isValidNetworkName("local") || isValidNetworkName("X") || !isValidNetworkName("ok-1") {
		t.Error("IsValidNetworkName logic wrong")
	}
}
