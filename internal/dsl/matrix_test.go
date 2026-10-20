package dsl_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

func TestMatrixPresetResolutionPreservesCaseOverridesAndSourceID(t *testing.T) {
	dir := t.TempDir()
	preset := []byte(`{"schemaVersion":"2","kind":"chain-preset","id":"selected","chain":"stablenet","values":{"scalar":1,"items":[1,2]}}`)
	path := filepath.Join(dir, "selected.json")
	if err := os.WriteFile(path, preset, 0600); err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"schemaVersion":"2","kind":"case","id":"case","chainPreset":{"extends":"old-missing-reference","id":"case-override","values":{"scalar":7}},"steps":[{"expect":"blockNumber","is":0}]}`)
	selected, results, err := dsl.CasesOnPreset([][]byte{raw}, dir, path)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := dsl.Parse(results[0])
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != "selected" || parsed.PresetID != "case-override" || parsed.EnvValues["scalar"] != float64(7) || len(parsed.EnvValues["items"].([]any)) != 2 {
		t.Fatalf("override or source changed: %+v %+v", selected, parsed)
	}
	var original map[string]any
	_ = json.Unmarshal(raw, &original)
	if original["chainPreset"].(map[string]any)["extends"] != "old-missing-reference" {
		t.Fatal("caller input mutated")
	}
}
