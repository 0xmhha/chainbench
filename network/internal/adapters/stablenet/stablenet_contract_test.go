package stablenet_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pelletier/go-toml/v2"

	"github.com/0xmhha/chainbench/network/internal/adapters/stablenet"
)

// updateGolden regenerates the files under testdata/golden/ instead of
// comparing. Invoke with: `go test ./internal/adapters/stablenet/... -update`.
// Review the diff before committing — the goldens are the pinned contract.
var updateGolden = flag.Bool("update", false, "regenerate golden files under testdata/golden/ instead of comparing")

// goldenDir is the on-disk location of the pinned adapter outputs, shared
// across all chain-type adapters in this sprint and future ones.
func goldenDir(t *testing.T) string {
	t.Helper()
	return filepath.Join("..", "testdata", "golden")
}

// TestContract_GenesisMatchesGolden runs GenerateGenesis against the shared
// fixtures and compares the rendered output to testdata/golden/stablenet-
// genesis.json via JSON deep-equal. Byte-level identity with the bash
// adapter is an explicit non-goal; semantic JSON equivalence is the contract.
func TestContract_GenesisMatchesGolden(t *testing.T) {
	in := decodeGenesisInput(t)
	in.OutputPath = filepath.Join(t.TempDir(), "genesis.json")

	if err := stablenet.New().GenerateGenesis(context.Background(), in); err != nil {
		t.Fatalf("GenerateGenesis: %v", err)
	}
	got, err := os.ReadFile(in.OutputPath)
	if err != nil {
		t.Fatalf("read generated: %v", err)
	}

	goldenPath := filepath.Join(goldenDir(t), "stablenet-genesis.json")
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir golden: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Logf("updated golden: %s", goldenPath)
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run `go test -update` to create)", goldenPath, err)
	}

	var gotParsed, wantParsed map[string]any
	if err := json.Unmarshal(got, &gotParsed); err != nil {
		t.Fatalf("generated is not valid JSON: %v", err)
	}
	if err := json.Unmarshal(want, &wantParsed); err != nil {
		t.Fatalf("golden is not valid JSON: %v", err)
	}
	if !reflect.DeepEqual(gotParsed, wantParsed) {
		gotPretty, _ := json.MarshalIndent(gotParsed, "", "  ")
		wantPretty, _ := json.MarshalIndent(wantParsed, "", "  ")
		t.Errorf("genesis contract mismatch (run `go test -update` to accept new output)\n--- want (golden) ---\n%s\n--- got ---\n%s",
			string(wantPretty), string(gotPretty))
	}
}

// writeDirPlaceholder is the sentinel substituted for the per-run WriteDir in
// generated TOML before the contract compare. GenerateToml inlines WriteDir
// into KeyStoreDir, so without normalization the golden would contain a
// t.TempDir() path that changes every run.
const writeDirPlaceholder = "__WRITE_DIR__"

// TestContract_TomlMatchesGolden renders config_node1.toml + config_node2.toml
// against the shared fixtures and compares each to its pinned golden. Parsing
// both sides via go-toml/v2 yields equal map[string]any trees under semantic
// TOML equivalence — whitespace, key ordering, and comments are ignored. The
// per-run WriteDir is normalized to writeDirPlaceholder so the goldens stay
// hermetic across runs and developers.
func TestContract_TomlMatchesGolden(t *testing.T) {
	dir := t.TempDir()
	in := decodeTomlInput(t, dir)

	if err := stablenet.New().GenerateToml(context.Background(), in); err != nil {
		t.Fatalf("GenerateToml: %v", err)
	}

	for i := 1; i <= in.TotalNodes; i++ {
		name := fmt.Sprintf("config_node%d.toml", i)
		generatedPath := filepath.Join(dir, name)
		raw, err := os.ReadFile(generatedPath)
		if err != nil {
			t.Fatalf("read generated %s: %v", name, err)
		}
		got := bytes.ReplaceAll(raw, []byte(dir), []byte(writeDirPlaceholder))

		goldenPath := filepath.Join(goldenDir(t), fmt.Sprintf("stablenet-config-node%d.toml", i))
		if *updateGolden {
			if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
				t.Fatalf("mkdir golden: %v", err)
			}
			if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
				t.Fatalf("write golden %s: %v", goldenPath, err)
			}
			t.Logf("updated golden: %s", goldenPath)
			continue
		}

		want, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("read golden %s: %v (run `go test -update` to create)", goldenPath, err)
		}

		var gotParsed, wantParsed map[string]any
		if err := toml.Unmarshal(got, &gotParsed); err != nil {
			t.Fatalf("generated %s is not valid TOML: %v", name, err)
		}
		if err := toml.Unmarshal(want, &wantParsed); err != nil {
			t.Fatalf("golden %s is not valid TOML: %v", goldenPath, err)
		}
		if !reflect.DeepEqual(gotParsed, wantParsed) {
			gotPretty, _ := json.MarshalIndent(gotParsed, "", "  ")
			wantPretty, _ := json.MarshalIndent(wantParsed, "", "  ")
			t.Errorf("%s contract mismatch (run `go test -update` to accept new output)\n--- want (golden) ---\n%s\n--- got ---\n%s",
				name, string(wantPretty), string(gotPretty))
		}
	}
}
