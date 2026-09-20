package session_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func TestWriteFileAtomic_NoLeftoverTmp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.json")
	if err := session.WriteFileAtomic(path, []byte(`{"x":1}`), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temp file left behind: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != `{"x":1}` {
		t.Fatalf("readback = %q, %v", got, err)
	}
	// Overwriting keeps the prior file readable throughout (rename is atomic).
	if err := session.WriteFileAtomic(path, []byte(`{"x":2}`), 0o644); err != nil {
		t.Fatalf("overwrite: %v", err)
	}
	got, _ = os.ReadFile(path)
	if string(got) != `{"x":2}` {
		t.Fatalf("after overwrite = %q", got)
	}
}

func TestRecord_Artifacts_RoundTrip(t *testing.T) {
	s, err := session.New(t.TempDir(), "run", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	r := s.Test(3, "gamma")
	want := session.TestArtifacts{Refs: []session.ArtifactRef{
		{Kind: "genesis", Ref: "environments/env-x/genesis/genesis.json"},
		{Kind: "config", Ref: "sha256:abc", Node: "node4"},
	}}
	r.Artifacts(want)

	data, err := os.ReadFile(filepath.Join(r.Dir(), "artifacts.json"))
	if err != nil {
		t.Fatalf("read artifacts.json: %v", err)
	}
	var got session.TestArtifacts
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got.Refs) != 2 || got.Refs[1].Node != "node4" || got.Refs[0].Kind != "genesis" {
		t.Fatalf("artifacts round-trip = %+v", got)
	}
}

func TestLoadDir_RoundTrip(t *testing.T) {
	s, err := session.New(t.TempDir(), "chainbench run x.json", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	s.Test(1, "one").Status(session.StatusPass)
	s.Test(2, "two").Status(session.StatusSkip)
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	res, err := session.LoadDir(s.Root())
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if res.Command != "chainbench run x.json" {
		t.Errorf("command = %q", res.Command)
	}
	if res.Summary.Pass != 1 || res.Summary.Skip != 1 {
		t.Errorf("summary = %+v", res.Summary)
	}
	if len(res.Tests) != 2 || res.Tests[0].ID != "one" || res.Tests[1].Status != "skip" {
		t.Errorf("tests = %+v", res.Tests)
	}
}
