package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetFilesHideInterruptedUploadsAndEnforceLimits(t *testing.T) {
	root := t.TempDir()
	s, err := OpenAssetFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Stage(strings.NewReader("12345"), 4); err == nil {
		t.Fatal("oversized input accepted")
	}
	if _, err = s.Stage(strings.NewReader(""), 4); err == nil {
		t.Fatal("empty input accepted")
	}
	stage, err := s.Stage(strings.NewReader("1234"), 4)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Discard(stage)
	ids, err := s.IDs()
	if err != nil || len(ids) != 0 {
		t.Fatal("unfinished upload visible", ids, err)
	}
	entries, err := os.ReadDir(filepath.Join(root, "assets"))
	if err != nil || len(entries) != 1 {
		t.Fatal("rejected upload not discarded", err)
	}
	id, err := NewAssetID()
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Publish(stage, id, []byte(`{"id":"receipt"}`), false); err != nil {
		t.Fatal(err)
	}
	raw, path, checksum, size, err := s.Read(id)
	if err != nil || len(raw) == 0 || path == "" || len(checksum) != 64 || size != 4 {
		t.Fatal("published asset missing", err)
	}
	for _, name := range []string{"payload", "receipt.json"} {
		info, err := os.Stat(filepath.Join(root, "assets", id, name))
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("asset permissions", err)
		}
	}
	if _, _, _, _, err = s.Read("../../outside"); !os.IsNotExist(err) {
		t.Fatal("path accepted", err)
	}
}
