package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStageFilesPreservesSourceAndBuildsRequiredContents(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.MkdirAll("presets/keys/node1", 0o755); err != nil {
		t.Fatal(err)
	}
	source := []byte("public test key fixture\n")
	if err := os.WriteFile("presets/keys/node1/keystore.json", source, 0o644); err != nil {
		t.Fatal(err)
	}
	stage := filepath.Join(root, "stage")
	if err := os.Mkdir(stage, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := stageFiles(stage); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		"kr-src/node1/keystore.json": string(source), "cb-kr-test/password": "pw\n",
		"cb-kr-test/metadata.json": "meta\n", "cb-kr-test/node1/keystore.json": "k1\n",
		"cb-sudo-test.txt": "root-only-test-content-12345\n",
	} {
		got, err := os.ReadFile(filepath.Join(stage, path))
		if err != nil || string(got) != want {
			t.Fatalf("%s: %q, %v", path, got, err)
		}
	}
	got, err := os.ReadFile("presets/keys/node1/keystore.json")
	if err != nil || string(got) != string(source) {
		t.Fatalf("source changed: %q %v", got, err)
	}
	log, err := os.ReadFile(filepath.Join(stage, "nodelog-test.log"))
	if err != nil || len(log) == 0 {
		t.Fatalf("missing log: %v", err)
	}
}

func TestStageFilesRefusesSymlinkInputs(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.MkdirAll("presets/keys", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("outside", []byte("unrelated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(root, "outside"), "presets/keys/key"); err != nil {
		t.Fatal(err)
	}
	if err := stageFiles(filepath.Join(root, "stage")); err == nil {
		t.Fatal("symlink fixture accepted")
	}
	got, err := os.ReadFile("outside")
	if err != nil || string(got) != "unrelated" {
		t.Fatalf("unrelated input changed: %q %v", got, err)
	}
}
