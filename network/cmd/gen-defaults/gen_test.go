package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// repoRoot is three levels up from this package (network/cmd/gen-defaults).
const repoRoot = "../../.."

// TestGeneratedFilesUpToDate guards against the SSOT-X1 source
// (network/schema/defaults.json) and the committed per-layer generated files
// drifting apart — e.g. someone edits defaults.json but forgets to run
// `go generate`, or hand-edits a generated file. Equivalent to
// `gen-defaults -check`, run as a normal unit test.
func TestGeneratedFilesUpToDate(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot, "network", "schema", "defaults.json"))
	if err != nil {
		t.Fatalf("read defaults.json: %v", err)
	}
	var d defaults
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("parse defaults.json: %v", err)
	}
	for rel, want := range render(d) {
		got, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			t.Errorf("%s: %v", rel, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s is stale; run `go generate ./...` in network/ to regenerate", rel)
		}
	}
}
