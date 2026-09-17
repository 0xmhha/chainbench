package session_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func TestList_And_ChainstatePaths(t *testing.T) {
	root := t.TempDir()

	// A real session with one environment holding a chainstate file.
	s, err := session.New(root, "run", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := s.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}
	env, err := s.NewEnvironment("ffffffffffff0000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	csPath := filepath.Join(env.ChainstateDir(), "chainstate.jsonl")
	if err := os.WriteFile(csPath, []byte(`{"seq":1}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A stray directory without session.json must be ignored.
	if err := os.MkdirAll(filepath.Join(root, "not-a-session"), 0o755); err != nil {
		t.Fatal(err)
	}

	// The id is root-relative, not the run alone: a run lives under the
	// directory of the process that produced it.
	id := session.IDFor(root, s.Root())
	ids, err := session.List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ids) != 1 || ids[0] != id {
		t.Fatalf("List = %v, want [%s]", ids, id)
	}

	if got := session.SessionFilePath(root, id); got != filepath.Join(s.Root(), "session.json") {
		t.Fatalf("SessionFilePath = %s", got)
	}

	paths, err := session.ChainstatePaths(root, id)
	if err != nil {
		t.Fatalf("ChainstatePaths: %v", err)
	}
	if len(paths) != 1 || paths[0] != csPath {
		t.Fatalf("ChainstatePaths = %v, want [%s]", paths, csPath)
	}
}

func TestList_MissingRootIsEmpty(t *testing.T) {
	ids, err := session.List(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("List missing root: %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("List = %v, want empty", ids)
	}
}
