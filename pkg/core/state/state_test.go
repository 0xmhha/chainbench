package state_test

import (
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/state"
)

func TestSaveLoadNodeSet(t *testing.T) {
	dir := t.TempDir()
	ns := node.NodeSet{
		Chain:        "stablenet",
		Network:      "local",
		Capabilities: []string{"rpc", "consensus"},
		Nodes: []node.Node{
			{Index: 1, Role: node.RoleValidator, Host: "127.0.0.1", RPCURL: "http://127.0.0.1:8501",
				Ports: node.Endpoints{P2P: 30301, HTTP: 8501}},
		},
	}
	if err := state.SaveNodeSet(dir, ns); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := state.LoadNodeSet(dir)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Chain != "stablenet" || len(got.Nodes) != 1 || got.Nodes[0].RPCURL != "http://127.0.0.1:8501" {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
	if _, err := state.LoadNodeSet(t.TempDir()); err == nil {
		t.Error("expected error loading from empty dir")
	}
}
