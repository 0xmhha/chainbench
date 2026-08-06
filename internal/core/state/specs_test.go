package state_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/state"
)

func TestSaveLoadNodeSpecs(t *testing.T) {
	dir := t.TempDir()
	specs := []driver.NodeSpec{
		{
			Index: 1, Role: node.RoleValidator, Host: "127.0.0.1",
			Binary: "gstable", DataDir: dir + "/node1", ConfigPath: dir + "/config_node1.toml",
			ConfigContent: []byte("[Node]\n"),
			Args:          []string{"--nodekey", "keys/node1/nodekey", "--unlock", "0xabc"},
			Ports:         node.Endpoints{P2P: 30301, HTTP: 8501},
		},
		{
			Index: 5, Role: node.RoleEndpoint, Host: "127.0.0.1",
			Binary: "gstable", DataDir: dir + "/node5", Ports: node.Endpoints{HTTP: 8505},
		},
	}
	if err := state.SaveNodeSpecs(dir, specs); err != nil {
		t.Fatalf("SaveNodeSpecs: %v", err)
	}
	got, err := state.LoadNodeSpecs(dir)
	if err != nil {
		t.Fatalf("LoadNodeSpecs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("loaded %d specs, want 2", len(got))
	}
	// Fields needed to relaunch must survive the round-trip.
	if got[0].Binary != "gstable" || got[0].Ports.HTTP != 8501 ||
		len(got[0].Args) != 4 || string(got[0].ConfigContent) != "[Node]\n" {
		t.Errorf("spec[0] round-trip lost fields: %+v", got[0])
	}
	if got[1].Index != 5 || got[1].Role != node.RoleEndpoint {
		t.Errorf("spec[1] round-trip wrong: %+v", got[1])
	}
}

func TestLoadNodeSpecs_Missing(t *testing.T) {
	if _, err := state.LoadNodeSpecs(t.TempDir()); err == nil {
		t.Error("LoadNodeSpecs on a dir with no nodespecs.json should error")
	}
}
