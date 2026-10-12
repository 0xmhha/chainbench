package chaincmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
)

func TestChainPresetPlanReportsScopesWithoutComposing(t *testing.T) {
	dir := t.TempDir()
	preset := filepath.Join(t.TempDir(), "preset.json")
	raw := `{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"stablenet","topology":{"bp":2,"en":1},"launch":{"en":{"nodiscover":true}}}`
	if err := os.WriteFile(preset, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	keys, err := filepath.Abs("../../../presets/keys")
	if err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "chain", "up", "--chain-preset", preset, "--plan", "--json", "--workspace-dir", dir, "--keys", keys, "--binary", "/selected/gstable")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Plan   app.ComposePlan `json:"plan"`
		Layout app.ChainUpIn   `json:"layout"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err, out)
	}
	if doc.Layout.BPCount != 2 || doc.Layout.ENCount != 1 || len(doc.Plan.Launch["en"]) != 1 || doc.Plan.Workspace != dir {
		t.Fatalf("missing preset layout or scope: %s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, "chain-record.json")); !os.IsNotExist(err) {
		t.Fatal("planning composed a network")
	}
}
