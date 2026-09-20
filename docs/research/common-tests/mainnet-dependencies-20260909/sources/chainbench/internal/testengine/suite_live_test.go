package testengine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/0xmhha/chainbench/internal/chains/stablenet" // register the stablenet plugin

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestSuite_Live_FullRun is the capstone: testengine.RunSuite composes a real
// 4-node stablenet through chainsetup — the one composition path since R4 —
// runs one spec against it, tears the network down, and reports.
//
// Gated on GSTABLE_BIN; CI skips it and stays green.
func TestSuite_Live_FullRun(t *testing.T) {
	bin := os.Getenv("GSTABLE_BIN")
	if bin == "" {
		t.Skip("set GSTABLE_BIN to a real gstable binary to run the full-suite live e2e")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("GSTABLE_BIN=%q: %v", bin, err)
	}
	// A short workspace keeps each node's IPC unix socket path under the
	// ~104-char limit.
	ws, err := os.MkdirTemp("/tmp", "cbe")
	if err != nil {
		t.Fatalf("mkdir workspace: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(ws) })

	spec, _ := json.Marshal(map[string]any{
		"schemaVersion": "1",
		"id":            "capstone",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{
			{"assert": "chainId", "expected": 8283},
			{"assert": "blockNumber", "compare": "GreaterOrEqual", "expected": "1"},
		},
	})
	specPath := filepath.Join(ws, "capstone.json")
	if err := os.WriteFile(specPath, spec, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	out, err := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{
		SpecPaths:  []string{specPath},
		DataDir:    ws,
		Binary:     bin,
		KeysDir:    filepath.Join(repoRoot(t), "keys", "preset"),
		WaitBlocks: 1,
	})
	if err != nil {
		t.Fatalf("RunSuite: %v (setup %v)", err, out.SetupSteps)
	}
	if out.Summary.Summary.Pass < 1 || out.Summary.Failed() {
		t.Fatalf("summary %+v (session %s)", out.Summary.Summary, out.SessionRoot)
	}
	if !out.Stopped {
		t.Error("the network was not torn down")
	}
	t.Logf("suite passed: %d test(s), session at %s", out.Summary.Summary.Pass, out.SessionRoot)
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found walking up from test dir")
		}
		dir = parent
	}
}
