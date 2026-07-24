package driver

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// fakeBinary writes an executable shell script that mimics a geth-family node:
// `init` succeeds (touching a marker), any other invocation exits 0 too.
func fakeBinary(t *testing.T, initExit int) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fakegeth")
	script := "#!/bin/sh\nif [ \"$1\" = \"init\" ]; then exit " + itoa(initExit) + "; fi\nexit 0\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	return "1"
}

func TestInitDatadir_Success(t *testing.T) {
	bin := fakeBinary(t, 0)
	dd := filepath.Join(t.TempDir(), "node1")
	if err := InitDatadir(context.Background(), bin, dd, "/tmp/genesis.json"); err != nil {
		t.Fatalf("InitDatadir: %v", err)
	}
	if _, err := os.Stat(dd); err != nil {
		t.Errorf("datadir not created: %v", err)
	}
}

func TestInitDatadir_Failure(t *testing.T) {
	bin := fakeBinary(t, 1) // init exits non-zero
	dd := filepath.Join(t.TempDir(), "node1")
	if err := InitDatadir(context.Background(), bin, dd, "/tmp/genesis.json"); err == nil {
		t.Error("expected error from failing init")
	}
}
