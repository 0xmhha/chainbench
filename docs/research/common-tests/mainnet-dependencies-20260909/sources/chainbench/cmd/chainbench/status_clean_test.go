package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusCmd(t *testing.T) {
	dir := t.TempDir()
	writeWorkspace(t, dir, "wbft", wsNode(dir, 1, "validator", 8501, 4321))
	out, err := run(t, "status", "--workspace-dir", dir)
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	if !strings.Contains(out, "chain: wbft") || !strings.Contains(out, "4321") {
		t.Errorf("status output unexpected:\n%s", out)
	}
}

func TestCleanCmd_GuardsAndRemoves(t *testing.T) {
	// a directory without chainbench artifacts is refused.
	safe := t.TempDir()
	if err := os.WriteFile(filepath.Join(safe, "unrelated.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, "clean", "--workspace-dir", safe); err == nil {
		t.Error("clean should refuse a non-chainbench dir")
	}
	if _, err := os.Stat(filepath.Join(safe, "unrelated.txt")); err != nil {
		t.Error("clean must not touch a refused dir")
	}

	// a real workspace (pid 0 = nothing to stop) is removed.
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	writeWorkspace(t, data, "wbft", wsNode(data, 1, "validator", 8501, 0))
	out, err := run(t, "clean", "--workspace-dir", data)
	if err != nil {
		t.Fatalf("clean: %v\n%s", err, out)
	}
	if _, err := os.Stat(data); !os.IsNotExist(err) {
		t.Errorf("data dir should be removed, stat err=%v", err)
	}
}
