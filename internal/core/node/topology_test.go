package node

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTmp(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "yaml")
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad_RolesSyncBootnode(t *testing.T) {
	p := writeTmp(t, `
chain: wemix
network: local
nodes:
  - { index: 1, role: bp,  sync_mode: full }
  - { index: 2, role: en,  sync_mode: full, bootnode: true }
  - { index: 3, role: en,  sync_mode: archive }
  - { index: 4, role: bp }
`)
	topo, err := LoadTopology(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if topo.Chain != "wemix" {
		t.Errorf("chain = %q", topo.Chain)
	}
	prod, ends := topo.Counts()
	if prod != 2 || ends != 2 {
		t.Errorf("counts = %d producers, %d endpoints; want 2,2", prod, ends)
	}
	if topo.BootnodeIndex() != 2 {
		t.Errorf("bootnode index = %d, want 2", topo.BootnodeIndex())
	}
	byIdx := map[int]TopologyEntry{}
	for _, n := range topo.Nodes {
		byIdx[n.Index] = n
	}
	if byIdx[1].NodeRole() != RoleValidator {
		t.Errorf("node1 role = %v, want validator", byIdx[1].NodeRole())
	}
	if byIdx[3].NodeRole() != RoleEndpoint || byIdx[3].EffectiveSyncMode() != "archive" {
		t.Errorf("node3 = %v/%s, want endpoint/archive", byIdx[3].NodeRole(), byIdx[3].EffectiveSyncMode())
	}
	if byIdx[4].EffectiveSyncMode() != "full" {
		t.Errorf("node4 default sync = %s, want full", byIdx[4].EffectiveSyncMode())
	}
}

func TestValidate_Errors(t *testing.T) {
	cases := map[string]Topology{
		"no chain":         {Nodes: []TopologyEntry{{Index: 1, Role: "bp"}}},
		"no nodes":         {Chain: "wbft"},
		"no producer":      {Chain: "wbft", Nodes: []TopologyEntry{{Index: 1, Role: "en"}}},
		"bad role":         {Chain: "wbft", Nodes: []TopologyEntry{{Index: 1, Role: "wizard"}}},
		"bad sync":         {Chain: "wbft", Nodes: []TopologyEntry{{Index: 1, Role: "bp", SyncMode: "warp"}}},
		"gap in indices":   {Chain: "wbft", Nodes: []TopologyEntry{{Index: 1, Role: "bp"}, {Index: 3, Role: "en"}}},
		"two bootnodes":    {Chain: "wbft", Nodes: []TopologyEntry{{Index: 1, Role: "bp", Bootnode: true}, {Index: 2, Role: "en", Bootnode: true}}},
		"duplicate index":  {Chain: "wbft", Nodes: []TopologyEntry{{Index: 1, Role: "bp"}, {Index: 1, Role: "en"}}},
		"index not from 1": {Chain: "wbft", Nodes: []TopologyEntry{{Index: 2, Role: "bp"}}},
	}
	for name, topo := range cases {
		if err := topo.Validate(); err == nil {
			t.Errorf("%s: expected validation error, got nil", name)
		}
	}
}

func TestValidate_OK(t *testing.T) {
	topo := Topology{Chain: "wbft", Nodes: []TopologyEntry{
		{Index: 1, Role: "validator"},
		{Index: 2, Role: "endpoint", SyncMode: "snap"},
	}}
	if err := topo.Validate(); err != nil {
		t.Fatalf("valid topology rejected: %v", err)
	}
}
