package mcp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestStartAndStopTool launches a small network with a fake node binary (init
// exits 0; run sleeps) and then stops it, exercising the real provision +
// InitDatadir + launch path (setup.Launch) and the stop path without a chain
// binary. The preset comes from the repo's keys/preset.
func TestStartAndStopTool(t *testing.T) {
	presetDir, err := filepath.Abs(filepath.Join("..", "..", "keys", "preset"))
	if err != nil || !dirExists(presetDir) {
		t.Skipf("preset not found at %s", presetDir)
	}
	fake := writeFakeBinary(t)
	dir := t.TempDir()
	s := newServer()

	text, isErr := callText(t, s, "chainbench_start", map[string]any{
		"chain": "stablenet", "binary": fake, "data_dir": dir,
		"validators": 2, "endpoints": 0, "keys_dir": presetDir,
	})
	// always attempt teardown, even on assertion failure.
	t.Cleanup(func() { callText(t, s, "chainbench_stop", map[string]any{"data_dir": dir}) })
	if isErr || !strings.Contains(text, "launched stablenet: 2 node") {
		t.Fatalf("start: err=%v text=%s", isErr, text)
	}
	if !strings.Contains(text, "pid=") {
		t.Errorf("start should report pids:\n%s", text)
	}
	// nodeset.json was persisted.
	if !fileExists(filepath.Join(dir, "nodeset.json")) {
		t.Error("nodeset.json not written")
	}

	stopText, isErr := callText(t, s, "chainbench_stop", map[string]any{"data_dir": dir})
	if isErr || !strings.Contains(stopText, "stopped 2 node") {
		t.Errorf("stop: err=%v text=%s", isErr, stopText)
	}
}

func writeFakeBinary(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "fakegeth")
	script := "#!/bin/sh\ncase \"$1\" in\n  init) exit 0 ;;\n  *) exec sleep 30 ;;\nesac\n"
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func dirExists(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
