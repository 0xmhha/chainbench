package filestore_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

func node(dataDir string) filestore.NodeInputs {
	return filestore.NodeInputs{
		DataDir: dataDir,
		Files: []filestore.File{
			{Path: "config.toml", Content: []byte("cfg"), Mode: 0o644},
			{Path: "genesis.json", Content: []byte("{}"), Mode: 0o644},
			{Path: "keystore/nodekey", Content: []byte("deadbeef"), Mode: 0o600},
		},
	}
}

func TestProvision_WritesFiles(t *testing.T) {
	base := t.TempDir()
	p := filestore.New(filestore.Local{})
	res, err := p.Provision(context.Background(), node(base))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if res.Written != 3 || res.Skipped != 0 {
		t.Fatalf("result = %+v, want 3 written 0 skipped", res)
	}
	if b, err := os.ReadFile(filepath.Join(base, "config.toml")); err != nil || string(b) != "cfg" {
		t.Fatalf("config = %q err=%v", b, err)
	}
	// Nested path created.
	fi, err := os.Stat(filepath.Join(base, "keystore", "nodekey"))
	if err != nil {
		t.Fatalf("nodekey missing: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("nodekey perm = %v, want 0600", fi.Mode().Perm())
	}
}

func TestProvision_ReusesIdenticalOverwritesDifferent(t *testing.T) {
	// Identical content already there: reused, never re-written.
	base := t.TempDir()
	if err := os.WriteFile(filepath.Join(base, "genesis.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := filestore.New(filestore.Local{}).Provision(context.Background(), node(base))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if res.Written != 2 || res.Skipped != 1 || res.Replaced != 0 {
		t.Fatalf("identical: result = %+v, want 2 written 1 skipped 0 replaced", res)
	}

	// Same name, DIFFERENT content: overwritten, never falsely reused (the E2
	// bug — the old exists-only check kept the stale file).
	base2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(base2, "genesis.json"), []byte("STALE"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err = filestore.New(filestore.Local{}).Provision(context.Background(), node(base2))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if res.Written != 2 || res.Replaced != 1 || res.Skipped != 0 {
		t.Fatalf("different: result = %+v, want 2 written 1 replaced 0 skipped", res)
	}
	if b, _ := os.ReadFile(filepath.Join(base2, "genesis.json")); string(b) != "{}" {
		t.Fatalf("stale file was not overwritten: %q", b)
	}
}

// fakeSink drives the upload-if-absent branch deterministically.
type fakeStore struct {
	present map[string]bool
	content map[string][]byte
	written []string
}

func (f *fakeStore) Exists(_ context.Context, path string) (bool, error) {
	return f.present[path], nil
}

func (f *fakeStore) Read(_ context.Context, path string) ([]byte, error) {
	b, ok := f.content[path]
	if !ok {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return b, nil
}

func (f *fakeStore) Checksum(_ context.Context, path string) (string, error) {
	b, ok := f.content[path]
	if !ok {
		return "", fmt.Errorf("not found: %s", path)
	}
	return filestore.Hash(b), nil
}

func (f *fakeStore) Write(_ context.Context, path string, content []byte, _ os.FileMode) error {
	f.written = append(f.written, path)
	if f.content == nil {
		f.content = map[string][]byte{}
	}
	f.content[path] = content
	return nil
}

func TestProvision_UploadIfAbsent(t *testing.T) {
	gp := filepath.Join("/remote/n1", "genesis.json")
	// Already on the server with the SAME content node() provisions ("{}").
	store := &fakeStore{
		present: map[string]bool{gp: true},
		content: map[string][]byte{gp: []byte("{}")},
	}
	res, err := filestore.New(store).Provision(context.Background(), node("/remote/n1"))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if res.Written != 2 || res.Skipped != 1 {
		t.Fatalf("result = %+v, want 2 written 1 skipped", res)
	}
	for _, w := range store.written {
		if filepath.Base(w) == "genesis.json" {
			t.Fatal("present identical remote file must not be re-uploaded")
		}
	}

	// Present but STALE: re-uploaded, not falsely reused.
	store2 := &fakeStore{
		present: map[string]bool{gp: true},
		content: map[string][]byte{gp: []byte("OLD")},
	}
	res, err = filestore.New(store2).Provision(context.Background(), node("/remote/n1"))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if res.Replaced != 1 {
		t.Fatalf("stale remote: result = %+v, want 1 replaced", res)
	}
}
