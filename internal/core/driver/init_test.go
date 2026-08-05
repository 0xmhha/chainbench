package driver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

// LocalDriver satisfies the Initializer capability.
var _ Initializer = (*LocalDriver)(nil)

func TestLocalDriver_InitDatadir(t *testing.T) {
	bin := fakeBinary(t, 0)
	dd := filepath.Join(t.TempDir(), "node1")
	d := NewLocalDriver()
	spec := NodeSpec{Index: 1, Binary: bin, DataDir: dd}
	if err := d.InitDatadir(context.Background(), spec, []byte(`{"config":{"chainId":1}}`)); err != nil {
		t.Fatalf("InitDatadir: %v", err)
	}
	// The genesis is placed inside the datadir for the init.
	b, err := os.ReadFile(filepath.Join(dd, "genesis.json"))
	if err != nil || !strings.Contains(string(b), "chainId") {
		t.Errorf("genesis not written into datadir: %v", err)
	}
}
