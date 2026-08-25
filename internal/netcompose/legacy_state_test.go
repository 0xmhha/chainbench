package netcompose_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/netcompose"
)

// TestOpen_ReadsTheLegacyServerConfigKey keeps a workspace composed before the
// serverConfig -> serverSet rename working: the recorded set path (and with it
// the docker localmap next to it) must survive the upgrade, and the next save
// writes only the new key.
func TestOpen_ReadsTheLegacyServerConfigKey(t *testing.T) {
	dir := t.TempDir()
	old := map[string]any{
		"chain":        "stablenet",
		"docker":       true,
		"serverConfig": "env/docker/build/server-set.yaml",
		"steps":        map[string]any{},
	}
	b, _ := json.Marshal(old)
	if err := os.WriteFile(filepath.Join(dir, "workspace.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := netcompose.Open(dir, nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if got := ws.State().ServerSet; got != "env/docker/build/server-set.yaml" {
		t.Fatalf("ServerSet = %q, want the legacy serverConfig value", got)
	}

	if err := ws.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	saved, err := os.ReadFile(filepath.Join(dir, "workspace.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(saved, &out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["serverConfig"]; ok {
		t.Error("the legacy key must not be written back")
	}
	if out["serverSet"] != "env/docker/build/server-set.yaml" {
		t.Errorf("serverSet = %v, want the migrated value", out["serverSet"])
	}
}
