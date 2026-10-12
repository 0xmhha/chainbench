package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func keySnapshotEngine(t *testing.T) (*WebChainEngine, string) {
	t.Helper()
	root, source := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(source, "nodekey"), []byte("private-key-snapshot-marker"), 0600); err != nil {
		t.Fatal(err)
	}
	documents, err := OpenDeploymentStore(filepath.Join(root, "documents"))
	if err != nil {
		t.Fatal(err)
	}
	return NewWebChainEngine(root, source, documents, nil, nil, nil), source
}

func TestWebKeySnapshotPreservesAcceptedBytesAcrossSourceEditAndRestart(t *testing.T) {
	e, source := keySnapshotEngine(t)
	snapshot, err := e.pinKeys(context.Background())
	if err != nil || len(snapshot.SHA256) != 64 {
		t.Fatalf("no immutable key snapshot: %+v %v", snapshot, err)
	}
	if err = os.WriteFile(filepath.Join(source, "nodekey"), []byte("replacement-key-marker"), 0600); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenDeploymentStore(filepath.Join(e.root, "documents"))
	if err != nil {
		t.Fatal(err)
	}
	restarted := NewWebChainEngine(e.root, source, reopened, nil, nil, nil)
	dir, err := restarted.materializeKeys(context.Background(), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "nodekey"))
	if err != nil || string(got) != "private-key-snapshot-marker" {
		t.Fatalf("accepted bytes changed: %v", err)
	}
	changed, err := e.pinKeys(context.Background())
	if err != nil || changed.SHA256 == snapshot.SHA256 {
		t.Fatal("changed source has the same accepted fingerprint")
	}
	info, err := os.Stat(filepath.Join(dir, "nodekey"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("materialized key is not private")
	}
	// The durable plan/job reference contains only a digest. Ciphertext storage
	// must not disclose the private material before execution materializes it.
	data, err := os.ReadFile(filepath.Join(e.root, "key-snapshots", snapshot.SHA256+".enc"))
	if err != nil || bytes.Contains(data, []byte("private-key-snapshot-marker")) {
		t.Fatal("snapshot is not encrypted")
	}
}

func TestWebKeySnapshotRejectsTamperingAndUnpinnedMaterial(t *testing.T) {
	e, _ := keySnapshotEngine(t)
	snapshot, err := e.pinKeys(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = e.materializeKeys(context.Background(), webKeySnapshot{}); err == nil {
		t.Fatal("unresolved key input accepted")
	}
	path := filepath.Join(e.root, "key-snapshots", snapshot.SHA256+".enc")
	if err = os.WriteFile(path, []byte("invalid encrypted input"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.materializeKeys(context.Background(), snapshot); err == nil {
		t.Fatal("tampered accepted snapshot used")
	}
}

func TestWebKeySnapshotRejectsMaterializedTampering(t *testing.T) {
	e, _ := keySnapshotEngine(t)
	snapshot, err := e.pinKeys(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	dir, err := e.materializeKeys(context.Background(), snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "nodekey"), []byte("changed-after-materialization"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.materializeKeys(context.Background(), snapshot); err == nil {
		t.Fatal("altered engine input accepted")
	}
}

func TestWebKeySnapshotRejectsLinksOversizeAndCancellation(t *testing.T) {
	for _, kind := range []string{"link", "oversize", "cancelled"} {
		t.Run(kind, func(t *testing.T) {
			e, source := keySnapshotEngine(t)
			ctx := context.Background()
			switch kind {
			case "link":
				if err := os.Symlink(filepath.Join(source, "nodekey"), filepath.Join(source, "linked")); err != nil {
					t.Fatal(err)
				}
			case "oversize":
				if err := os.WriteFile(filepath.Join(source, "oversize"), []byte(strings.Repeat("x", (1<<20)+1)), 0600); err != nil {
					t.Fatal(err)
				}
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if _, err := e.pinKeys(ctx); err == nil {
				t.Fatal("unsafe or cancelled key capture accepted")
			}
		})
	}
}

func TestWebChainExecutionRejectsUnpinnedKeysBeforeEffects(t *testing.T) {
	e, _ := keySnapshotEngine(t)
	p := webChainPayload{Input: WebPlanInput{Operation: "chain.setup"}, Set: DeploymentDocument{DeploymentDocumentInput: deploymentTestSet()}, Config: DeploymentDocument{DeploymentDocumentInput: deploymentTestConfig()}, Arguments: webChainArguments{AssetID: "fixture"}}
	e.assets["fixture"] = ManifestBinary{ID: "fixture", Path: "/fixture/native"}
	phases := 0
	_, err := e.execute(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, p, func(WebJobPhase) error { phases++; return errors.New("stop before an engine effect") })
	if err == nil || phases != 0 {
		t.Fatalf("unpinned input reached execution phases: %d %v", phases, err)
	}
}
