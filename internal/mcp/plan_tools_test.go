package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	_ "github.com/0xmhha/chainbench/internal/chains/all"
)

func TestToolsExposeReadOnlyCompositionPlans(t *testing.T) {
	preset := `{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"stablenet","topology":{"bp":2,"en":1},"launch":{"en":{"nodiscover":true}}}`
	for _, tool := range []Tool{runTool(), chainUpTool()} {
		t.Run(tool.Name, func(t *testing.T) {
			props := tool.InputSchema["properties"].(map[string]any)
			if props["plan"] == nil {
				t.Fatal("tool has no read-only plan input")
			}
			dir := t.TempDir()
			keys, err := filepath.Abs("../../presets/keys")
			if err != nil {
				t.Fatal(err)
			}
			args := map[string]any{"plan": true, "dataDir": dir, "workspaceDir": dir, "keysDir": keys, "binary": "/selected/gstable", "suite": true}
			if tool.Name == "chainbench_run" {
				args["spec"] = `{"schemaVersion":"2","kind":"case","id":"c","chainPreset":` + preset + `,"steps":[{"expect":"blockNumber","is":0}]}`
			} else {
				path := filepath.Join(t.TempDir(), "preset.json")
				if err = os.WriteFile(path, []byte(preset), 0600); err != nil {
					t.Fatal(err)
				}
				args["chainPreset"] = path
			}
			out, err := tool.Handler(context.Background(), args)
			if err != nil {
				t.Fatal(err)
			}
			if tool.Name == "chainbench_run" {
				var doc []app.SuitePlan
				if err = json.Unmarshal([]byte(out), &doc); err != nil || len(doc) != 1 || doc[0].Layout.ENCount != 1 {
					t.Fatal(out, err)
				}
			} else {
				var doc app.PresetPlan
				if err = json.Unmarshal([]byte(out), &doc); err != nil || doc.Layout.BPCount != 2 || len(doc.Plan.Launch["en"]) != 1 {
					t.Fatal(out, err)
				}
			}
			if _, err = os.Stat(filepath.Join(dir, "chain-record.json")); !os.IsNotExist(err) {
				t.Fatal("MCP planning composed nodes")
			}
		})
	}
}
