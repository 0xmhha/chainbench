package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
)

func TestInlineToolSpecsResolveNamedPresetsAndOverrides(t *testing.T) {
	keys, err := filepath.Abs("../../presets/keys")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"explicit baseDir", "working directory", "relative context"} {
		t.Run(mode, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Mkdir(filepath.Join(dir, "chain-preset"), 0700); err != nil {
				t.Fatal(err)
			}
			preset := `{"schemaVersion":"2","kind":"chain-preset","id":"shared","chain":"stablenet","topology":{"bp":2,"en":1}}`
			if err := os.WriteFile(filepath.Join(dir, "chain-preset/shared.json"), []byte(preset), 0600); err != nil {
				t.Fatal(err)
			}
			specs := []string{
				`{"schemaVersion":"2","kind":"case","id":"reference","chainPreset":"shared","steps":[{"expect":"blockNumber","is":1}]}`,
				`{"schemaVersion":"2","kind":"case","id":"override","chainPreset":{"extends":"shared","topology":{"bp":3}},"steps":[{"expect":"blockNumber","is":1}]}`,
			}
			args := map[string]any{"specs": specs, "plan": true, "dataDir": filepath.Join(dir, "workspace"), "keysDir": keys, "binary": "/selected/gstable"}
			switch mode {
			case "explicit baseDir":
				args["baseDir"] = dir
			case "relative context":
				child := filepath.Join(dir, "nested")
				if err := os.Mkdir(child, 0700); err != nil {
					t.Fatal(err)
				}
				t.Chdir(child)
				args["baseDir"] = "."
			default:
				t.Chdir(dir)
			}
			output, err := runTool().Handler(context.Background(), args)
			if err != nil {
				t.Fatal(err)
			}
			var plans []app.SuitePlan
			if err := json.Unmarshal([]byte(output), &plans); err != nil || len(plans) != 2 {
				t.Fatal(err, output)
			}
			for i, plan := range plans {
				if plan.Layout.BPCount != 2+i || plan.Layout.ENCount != 1 {
					t.Fatalf("preset or override lost: %+v", plan.Layout)
				}
			}
			for _, tool := range []Tool{runTool(), validateTool()} {
				if tool.InputSchema["properties"].(map[string]any)["baseDir"] == nil {
					t.Fatalf("%s lacks preset context input", tool.Name)
				}
			}
			args["specs"] = append(specs, `{"schemaVersion":"2","kind":"case","id":"missing","chainPreset":"not-declared","steps":[{"expect":"blockNumber","is":1}]}`)
			validation, err := validateTool().Handler(context.Background(), args)
			if err != nil || !strings.Contains(validation, "valid=2 invalid=1") || !strings.Contains(validation, "not-declared") {
				t.Fatalf("validation did not preserve per-spec results: %v %s", err, validation)
			}
			if _, err := os.Stat(filepath.Join(dir, "workspace/chain-record.json")); !os.IsNotExist(err) {
				t.Fatal("read-only plan/validation created a network")
			}
		})
	}
}
