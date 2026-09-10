package chainsetup

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testsupport"
)

// TestLive_MaterializeKeyringDownloadsAServerKeyring verifies the srv:// keyring
// path end to end against the docker fleet: a key set placed on a server is
// downloaded to a local directory by the keys step, and the downloaded ring
// loads — so a node can sign test transactions with keys at that local path.
// Set up first:
//
//	docker exec chainbench-server1 mkdir -p /data/chainbench/keys
//	docker cp keys/preset chainbench-server1:/data/chainbench/keys/kr-src
//	CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build go test ./internal/chainsetup -run TestLive_MaterializeKeyring -v
func TestLive_MaterializeKeyringDownloadsAServerKeyring(t *testing.T) {
	build := testsupport.ServersBuildDir(t)
	dir := t.TempDir()
	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.ServerSet = filepath.Join(build, "server-set.yaml")
	w.state.Docker = true
	w.state.Target = resource.Spec{DataRoot: "/data/chainbench"}
	w.state.KeysDir = "srv://server1/data/chainbench/keys/kr-src"

	if err := w.materializeKeyring(context.Background()); err != nil {
		t.Fatalf("materializeKeyring: %v", err)
	}

	// KeysDir now points at a local copy, not the srv:// reference.
	wantLocal := filepath.Join(dir, "downloaded-keys")
	if w.state.KeysDir != wantLocal {
		t.Fatalf("KeysDir = %q, want the local download dir %q", w.state.KeysDir, wantLocal)
	}
	// The downloaded ring loads — the same read a node's signing uses.
	preset, err := store.LoadPreset(wantLocal)
	if err != nil {
		t.Fatalf("downloaded keyring does not load: %v", err)
	}
	if len(preset.Nodes) == 0 {
		t.Fatal("downloaded keyring has no node identities")
	}
}
