package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestResourceReadToolsMirrorCommandObservations(t *testing.T) {
	s := Default("reads", "test")
	for _, name := range []string{"chainbench_resource_inspect", "chainbench_resource_access"} {
		tool, ok := s.tools[name]
		if !ok || !tool.ReadOnly {
			t.Fatalf("missing read-only tool %s", name)
		}
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "data")
	if err := os.WriteFile(file, []byte("immutable"), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := s.tools["chainbench_resource_inspect"].Handler(context.Background(), map[string]any{"request": map[string]any{"target": map[string]any{"dataRoot": dir}, "inspect": true, "checksums": []any{file}}})
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["target"] == nil || doc["checksums"] == nil {
		t.Fatal(out)
	}
}
