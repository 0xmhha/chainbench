package app

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNativeAssetProbeOutputCannotBypassBoundThroughCopy(t *testing.T) {
	var output manifestProbeOutput
	// Wrapping the reader disables its WriterTo shortcut. io.Copy must not find
	// an unbounded promoted bytes.Buffer.ReadFrom method on the destination.
	reader := struct{ io.Reader }{strings.NewReader(strings.Repeat("x", (1<<20)+1))}
	if _, err := io.Copy(&output, reader); err == nil {
		t.Fatal("native output limit bypassed")
	}
	if len(output.Bytes()) > 1<<20 {
		t.Fatal("oversized native output buffered")
	}
}

func TestAssetStorageFailureDoesNotExposePrivatePath(t *testing.T) {
	root := t.TempDir()
	s, err := OpenManifestStore(root)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(filepath.Join(root, "assets"), filepath.Join(root, "assets-unavailable")); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Assets(); err == nil || strings.Contains(err.Error(), root) {
		t.Fatal("private asset storage path exposed", err)
	}
}

func TestRegisteredAssetRejectsReplacementAndTraversal(t *testing.T) {
	root := t.TempDir()
	s, err := OpenManifestStore(root)
	if err != nil {
		t.Fatal(err)
	}
	actor := DeploymentActor{ID: "alice", Role: "operator"}
	for _, name := range []string{"../config.json", `sub\config.json`, ".", "bad\nname.json"} {
		if _, err = s.UploadAsset(context.Background(), actor, "configuration", name, strings.NewReader(`{"chainId":1}`)); err == nil {
			t.Fatal("unsafe name accepted", name)
		}
	}
	item, err := s.UploadAsset(context.Background(), actor, "configuration", "node.json", strings.NewReader(`{"chainId":1}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.BinaryAsset(item.ID); err == nil {
		t.Fatal("non-binary executable selection accepted")
	}
	if _, err = s.Asset("../../outside"); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatal("invalid ID exposed storage", err)
	}
	path := filepath.Join(root, "assets", item.ID, "payload")
	if err = os.WriteFile(path, []byte(`{"chainId":2}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Asset(item.ID); err == nil {
		t.Fatal("changed payload accepted")
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside.json")
	_ = os.WriteFile(outside, []byte(`{"chainId":1}`), 0600)
	if err = os.Symlink(outside, path); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Asset(item.ID); err == nil {
		t.Fatal("linked payload accepted")
	}
	if raw, err := os.ReadFile(outside); err != nil || string(raw) != `{"chainId":1}` {
		t.Fatal("outside file changed")
	}
	if _, err = s.UploadAsset(context.Background(), DeploymentActor{ID: "viewer", Role: "viewer"}, "configuration", "node.json", strings.NewReader(`{}`)); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("viewer upload", err)
	}
}

func TestAssetFormatsPreserveExactBytesAndRejectCredentials(t *testing.T) {
	s, err := OpenManifestStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	actor := DeploymentActor{ID: "alice", Role: "operator"}
	for _, tt := range []struct{ kind, name, data string }{{"configuration", "node.yaml", "chainId: 91\n"}, {"template", "genesis.json", `{"config":{"chainId":91}}`}, {"material", "contract.abi", "[]"}, {"material", "code.bin", "0x60006000"}} {
		item, err := s.UploadAsset(context.Background(), actor, tt.kind, tt.name, strings.NewReader(tt.data))
		if err != nil {
			t.Fatal(err)
		}
		if item.Bytes != int64(len(tt.data)) {
			t.Fatal("input bytes transformed")
		}
	}
	for _, data := range []string{`{"key_file":"private.pem"}`, `{"nested":[{"privateKey":"sensitive"}]}`, "ssh:\n  password: private\n", `{"crypto":{"ciphertext":"encrypted-keystore"}}`} {
		name := "secret.json"
		if strings.HasPrefix(data, "ssh:") {
			name = "secret.yaml"
		}
		if _, err = s.UploadAsset(context.Background(), actor, "configuration", name, strings.NewReader(data)); err == nil {
			t.Fatal("credential accepted", data)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = s.UploadAsset(ctx, actor, "template", "genesis.json", strings.NewReader(`{}`)); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled upload published", err)
	}
}
