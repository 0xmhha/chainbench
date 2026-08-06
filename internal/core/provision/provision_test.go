package provision_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/provision"
)

func node(dataDir string) provision.NodeInputs {
	return provision.NodeInputs{
		DataDir: dataDir,
		Files: []provision.File{
			{Path: "config.toml", Content: []byte("cfg"), Mode: 0o644},
			{Path: "genesis.json", Content: []byte("{}"), Mode: 0o644},
			{Path: "keystore/nodekey", Content: []byte("deadbeef"), Mode: 0o600},
		},
	}
}

func TestProvision_WritesFiles(t *testing.T) {
	base := t.TempDir()
	p := provision.New(provision.LocalFileSink{})
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

func TestProvision_SkipsExisting(t *testing.T) {
	base := t.TempDir()
	// Pre-create genesis.json so it is reused, not overwritten.
	if err := os.WriteFile(filepath.Join(base, "genesis.json"), []byte("EXISTING"), 0o644); err != nil {
		t.Fatal(err)
	}
	p := provision.New(provision.LocalFileSink{})
	res, err := p.Provision(context.Background(), node(base))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if res.Written != 2 || res.Skipped != 1 {
		t.Fatalf("result = %+v, want 2 written 1 skipped", res)
	}
	if b, _ := os.ReadFile(filepath.Join(base, "genesis.json")); string(b) != "EXISTING" {
		t.Fatalf("existing file was overwritten: %q", b)
	}
}

// fakeSink drives the upload-if-absent branch deterministically.
type fakeSink struct {
	present map[string]bool
	written []string
}

func (f *fakeSink) Exists(_ context.Context, path string) (bool, error) {
	return f.present[path], nil
}

func (f *fakeSink) Write(_ context.Context, path string, _ []byte, _ os.FileMode) error {
	f.written = append(f.written, path)
	return nil
}

func TestProvision_UploadIfAbsent(t *testing.T) {
	sink := &fakeSink{present: map[string]bool{
		filepath.Join("/remote/n1", "genesis.json"): true, // already on the server
	}}
	p := provision.New(sink)
	res, err := p.Provision(context.Background(), node("/remote/n1"))
	if err != nil {
		t.Fatalf("Provision: %v", err)
	}
	if res.Written != 2 || res.Skipped != 1 {
		t.Fatalf("result = %+v, want 2 written 1 skipped", res)
	}
	for _, w := range sink.written {
		if filepath.Base(w) == "genesis.json" {
			t.Fatal("present remote file must not be re-uploaded")
		}
	}
}
