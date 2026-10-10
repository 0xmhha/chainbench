package session

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func assetInputFixture(t *testing.T) (*AssetFiles, string, string, []byte) {
	t.Helper()
	s, err := OpenAssetFiles(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("{\n\"config\":{\"chainId\":9410}\n}\n")
	stage, err := s.Stage(bytes.NewReader(raw), 16<<20)
	if err != nil {
		t.Fatal(err)
	}
	id, err := NewAssetID()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Publish(stage, id, []byte(`{}`), false); err != nil {
		t.Fatal(err)
	}
	return s, id, stage.Checksum, raw
}

func TestAssetInputConcurrentPublicationPreservesBytes(t *testing.T) {
	s, id, checksum, raw := assetInputFixture(t)
	fingerprint := strings.Repeat("a", 64)
	var wg sync.WaitGroup
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			path, err := s.MaterializeFile(context.Background(), id, checksum, fingerprint)
			if err != nil {
				t.Error(err)
				return
			}
			got, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(got, raw) {
				t.Error("snapshot changed bytes", err)
			}
			info, err := os.Stat(path)
			if err != nil || info.Mode().Perm() != 0600 {
				t.Error("snapshot is not private", err)
			}
		}()
	}
	wg.Wait()
}

func TestAssetInputRefusesChangesAndLinksWithoutRepair(t *testing.T) {
	for _, target := range []string{"original", "snapshot", "link", "parent-link"} {
		t.Run(target, func(t *testing.T) {
			s, id, checksum, raw := assetInputFixture(t)
			fingerprint := strings.Repeat("b", 64)
			path, err := s.MaterializeFile(context.Background(), id, checksum, fingerprint)
			if err != nil {
				t.Fatal(err)
			}
			changed := []byte(`{"config":{"chainId":1}}`)
			switch target {
			case "original":
				path = filepath.Join(s.root, id, "payload")
			case "link":
				outside := filepath.Join(t.TempDir(), "outside.json")
				if err = os.WriteFile(outside, raw, 0600); err != nil {
					t.Fatal(err)
				}
				if err = os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err = os.Symlink(outside, path); err != nil {
					t.Fatal(err)
				}
			case "parent-link":
				parent := filepath.Dir(filepath.Dir(path))
				if err = os.RemoveAll(parent); err != nil {
					t.Fatal(err)
				}
				outside := t.TempDir()
				if err = os.Symlink(outside, parent); err != nil {
					t.Fatal(err)
				}
				if _, err = s.MaterializeFile(context.Background(), id, checksum, fingerprint); err == nil {
					t.Fatal("linked snapshot parent accepted")
				}
				entries, err := os.ReadDir(outside)
				if err != nil || len(entries) != 0 {
					t.Fatal("linked parent caused outside writes", err)
				}
				return
			}
			if target != "link" {
				if err = os.WriteFile(path, changed, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err = s.MaterializeFile(context.Background(), id, checksum, fingerprint); err == nil {
				t.Fatal("changed or linked input accepted")
			}
			if target != "link" {
				got, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(got, changed) {
					t.Fatal("changed input was silently repaired", err)
				}
			}
		})
	}
}

func TestAssetInputCancellationHasNoSnapshotEffects(t *testing.T) {
	s, id, checksum, _ := assetInputFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.MaterializeFile(ctx, id, checksum, strings.Repeat("c", 64)); err != context.Canceled {
		t.Fatal("cancelled input accepted", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(s.root), "asset-material")); !os.IsNotExist(err) {
		t.Fatal("cancelled input created snapshot", err)
	}
}
