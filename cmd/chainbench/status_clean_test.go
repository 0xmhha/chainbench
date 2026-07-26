package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusCmd(t *testing.T) {
	dir := t.TempDir()
	ns := `{"chain":"wbft","network":"local","nodes":[{"index":1,"role":"validator","rpc_url":"http://127.0.0.1:8501","pid":4321}]}`
	if err := os.WriteFile(filepath.Join(dir, "nodeset.json"), []byte(ns), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "status", "--data-dir", dir)
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
	if _, err := run(t, "clean", "--data-dir", safe); err == nil {
		t.Error("clean should refuse a non-chainbench dir")
	}
	if _, err := os.Stat(filepath.Join(safe, "unrelated.txt")); err != nil {
		t.Error("clean must not touch a refused dir")
	}

	// a real data dir (has nodeset.json, pid 0 = nothing to stop) is removed.
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	if err := os.MkdirAll(data, 0o755); err != nil {
		t.Fatal(err)
	}
	ns := `{"chain":"wbft","network":"local","nodes":[{"index":1,"role":"validator","rpc_url":"http://x","pid":0}]}`
	if err := os.WriteFile(filepath.Join(data, "nodeset.json"), []byte(ns), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "clean", "--data-dir", data)
	if err != nil {
		t.Fatalf("clean: %v\n%s", err, out)
	}
	if _, err := os.Stat(data); !os.IsNotExist(err) {
		t.Errorf("data dir should be removed, stat err=%v", err)
	}
}
