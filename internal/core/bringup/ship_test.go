package bringup

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/driver"
)

// recordingFP records the paths a shipIdentities call provisions.
type recordingFP struct{ paths []string }

func (r *recordingFP) ProvisionFile(_ context.Context, path string, _ []byte, _ fs.FileMode) error {
	r.paths = append(r.paths, path)
	return nil
}

func TestShipIdentities(t *testing.T) {
	// Build a fake local key set: password, node1 (validator: nodekey + keystore),
	// node2 (endpoint: nodekey only).
	keysAbs := t.TempDir()
	write := func(p string) {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(keysAbs, "password"))
	write(filepath.Join(keysAbs, "node1", "nodekey"))
	write(filepath.Join(keysAbs, "node1", "keystore", "UTC--key"))
	write(filepath.Join(keysAbs, "node2", "nodekey"))

	keyBase := "/remote/keys"
	nodes := []driver.NodeSpec{{Index: 1}, {Index: 2}}
	fp := &recordingFP{}
	if err := shipIdentities(context.Background(), fp, keysAbs, keyBase, nodes); err != nil {
		t.Fatalf("shipIdentities: %v", err)
	}

	sort.Strings(fp.paths)
	want := []string{
		"/remote/keys/node1/keystore/UTC--key",
		"/remote/keys/node1/nodekey",
		"/remote/keys/node2/nodekey",
		"/remote/keys/password",
	}
	if len(fp.paths) != len(want) {
		t.Fatalf("shipped %v, want %v", fp.paths, want)
	}
	for i := range want {
		if fp.paths[i] != want[i] {
			t.Errorf("path %d = %q, want %q", i, fp.paths[i], want[i])
		}
	}
}

// ensure the fake satisfies the interface used by Launch.
var _ driver.FileProvisioner = (*recordingFP)(nil)
