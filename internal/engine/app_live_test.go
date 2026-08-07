package engine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/engine"

	_ "github.com/0xmhha/chainbench/internal/chains/stablenet" // register the stablenet plugin
)

// TestEngine_Live_FullRun is the capstone: it composes a runnable Engine with
// NewLocalEngine and drives one spec through Engine.Run against a real gstable
// binary — build a 4-node network, run the spec's tx step and assertions, tear
// the network down, and save the session. It asserts the recorded verdict is a
// pass, exercising every wired component in one call.
//
// Gated on GSTABLE_BIN; CI skips it and stays green.
func TestEngine_Live_FullRun(t *testing.T) {
	bin := os.Getenv("GSTABLE_BIN")
	if bin == "" {
		t.Skip("set GSTABLE_BIN to a real gstable binary to run the full-engine live e2e")
	}
	if _, err := os.Stat(bin); err != nil {
		t.Fatalf("GSTABLE_BIN=%q: %v", bin, err)
	}

	// A short artifact root keeps each node's IPC unix socket path (nested under
	// the session/env datadirs) under the ~104-char limit.
	artifactRoot, err := os.MkdirTemp("/tmp", "e")
	if err != nil {
		t.Fatalf("mkdir artifact root: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(artifactRoot) })

	eng, err := engine.NewLocalEngine(engine.LocalConfig{
		Chain:        "stablenet",
		Binary:       bin,
		KeysDir:      filepath.Join(repoRoot(t), "keys", "preset"),
		ArtifactRoot: artifactRoot,
		Validators:   4,
		Clock:        func() time.Time { return time.Unix(0, 0).UTC() },
	})
	if err != nil {
		t.Fatalf("NewLocalEngine: %v", err)
	}

	spec, _ := json.Marshal(map[string]any{
		"schemaVersion": "1",
		"id":            "capstone",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{
			{"assert": "chainId", "expected": 8283},
			{"assert": "blockNumber", "expected": 1},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	root, err := eng.Run(ctx, [][]byte{spec})
	if err != nil {
		t.Fatalf("Engine.Run: %v", err)
	}

	// Read the saved session and assert the spec passed.
	data, err := os.ReadFile(filepath.Join(root, "session.json"))
	if err != nil {
		t.Fatalf("read session.json: %v", err)
	}
	var doc struct {
		Tests []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"tests"`
		Summary struct {
			Pass int `json:"pass"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse session.json: %v", err)
	}
	if doc.Summary.Pass < 1 {
		t.Fatalf("expected at least one passing test, got summary %+v (tests %+v)", doc.Summary, doc.Tests)
	}
	t.Logf("Engine.Run passed: %d test(s), session at %s", doc.Summary.Pass, root)
}
