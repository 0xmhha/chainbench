package state_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/state"
	"github.com/0xmhha/chainbench/network/schema"
)

// setupStateDir creates a temporary state/ directory with fixture files and
// returns its path.
func setupStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	copy := func(src, dst string) {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, dst), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	copy("testdata/pids-default.json", "pids.json")
	copy("testdata/profile-default.yaml", "current-profile.yaml")
	return dir
}

func TestLoadActive_HappyPath(t *testing.T) {
	dir := setupStateDir(t)
	net, err := state.LoadActive(state.LoadActiveOptions{StateDir: dir, Name: "local"})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if net.Name != "local" {
		t.Errorf("name: got %q, want local", net.Name)
	}
	if string(net.ChainType) != "stablenet" {
		t.Errorf("chain_type: got %q, want stablenet (default)", net.ChainType)
	}
	if net.ChainId != 8283 {
		t.Errorf("chain_id: got %d, want 8283", net.ChainId)
	}
	if len(net.Nodes) != 2 {
		t.Fatalf("nodes: got %d, want 2", len(net.Nodes))
	}
	// Nodes must be sorted by numeric key ascending.
	if net.Nodes[0].Id != "node1" {
		t.Errorf("nodes[0].id: got %q, want node1", net.Nodes[0].Id)
	}
	if net.Nodes[1].Id != "node5" {
		t.Errorf("nodes[1].id: got %q, want node5", net.Nodes[1].Id)
	}
	// Provider and URLs.
	if string(net.Nodes[0].Provider) != "local" {
		t.Errorf("provider: got %q", net.Nodes[0].Provider)
	}
	if net.Nodes[0].Http != "http://127.0.0.1:8501" {
		t.Errorf("http: got %q", net.Nodes[0].Http)
	}
	if net.Nodes[0].Ws == nil || *net.Nodes[0].Ws != "ws://127.0.0.1:9501" {
		t.Errorf("ws: got %v", net.Nodes[0].Ws)
	}
	// Role mapping.
	if net.Nodes[0].Role == nil || string(*net.Nodes[0].Role) != "validator" {
		t.Errorf("role: got %v", net.Nodes[0].Role)
	}
	if net.Nodes[1].Role == nil || string(*net.Nodes[1].Role) != "endpoint" {
		t.Errorf("role: got %v", net.Nodes[1].Role)
	}
}

func TestLoadActive_OutputValidatesAgainstSchema(t *testing.T) {
	dir := setupStateDir(t)
	net, err := state.LoadActive(state.LoadActiveOptions{StateDir: dir, Name: "local"})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	raw, err := json.Marshal(net)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := schema.ValidateBytes("network", raw); err != nil {
		t.Fatalf("schema validation: %v\nraw: %s", err, raw)
	}
}

func TestLoadActive_DefaultsStateDirAndName(t *testing.T) {
	dir := setupStateDir(t)
	// StateDir provided but Name empty -> default "local".
	net, err := state.LoadActive(state.LoadActiveOptions{StateDir: dir})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if net.Name != "local" {
		t.Errorf("name default: got %q", net.Name)
	}
}

func TestLoadActive_MissingPIDs(t *testing.T) {
	dir := t.TempDir()
	// profile exists, pids.json missing
	data, _ := os.ReadFile("testdata/profile-default.yaml")
	_ = os.WriteFile(filepath.Join(dir, "current-profile.yaml"), data, 0o644)
	_, err := state.LoadActive(state.LoadActiveOptions{StateDir: dir, Name: "local"})
	if err == nil {
		t.Fatal("expected error when pids.json is missing")
	}
}

func TestLoadActive_MissingProfile(t *testing.T) {
	dir := t.TempDir()
	data, _ := os.ReadFile("testdata/pids-default.json")
	_ = os.WriteFile(filepath.Join(dir, "pids.json"), data, 0o644)
	_, err := state.LoadActive(state.LoadActiveOptions{StateDir: dir, Name: "local"})
	if err == nil {
		t.Fatal("expected error when profile is missing")
	}
}
