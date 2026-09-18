package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// makeSession creates a session dir under root with a session.json and sets its
// mtime, returning the id. An incomplete session (no session.json) is created by
// passing complete=false.
func makeSession(t *testing.T, root, id string, age time.Duration, complete bool) {
	t.Helper()
	dir := filepath.Join(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if complete {
		f := filepath.Join(dir, "session.json")
		if err := os.WriteFile(f, []byte(`{"id":"`+id+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		when := time.Now().Add(-age)
		if err := os.Chtimes(f, when, when); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCleanSessions_OlderThan(t *testing.T) {
	root := t.TempDir()
	makeSession(t, root, "UTC-old", 10*24*time.Hour, true)
	makeSession(t, root, "UTC-new", 1*time.Hour, true)
	makeSession(t, root, "UTC-running", 0, false) // no session.json: in progress

	out, err := run(t, "clean", "--artifact-root", root, "--older-than", "7d")
	if err != nil {
		t.Fatalf("clean: %v\n%s", err, out)
	}
	if _, err := os.Stat(filepath.Join(root, "UTC-old")); !os.IsNotExist(err) {
		t.Fatal("old session should be removed")
	}
	if _, err := os.Stat(filepath.Join(root, "UTC-new")); err != nil {
		t.Fatal("recent session must be kept")
	}
	if _, err := os.Stat(filepath.Join(root, "UTC-running")); err != nil {
		t.Fatal("in-progress session (no session.json) must be preserved")
	}
	if !strings.Contains(out, "removed session UTC-old") {
		t.Fatalf("output should name the removed session:\n%s", out)
	}
}

func TestCleanSessions_KeepLast(t *testing.T) {
	root := t.TempDir()
	// IDs are sorted ascending; keep-last keeps the lexicographically largest.
	for _, id := range []string{"UTC-1", "UTC-2", "UTC-3"} {
		makeSession(t, root, id, time.Hour, true)
	}
	if _, err := run(t, "clean", "--artifact-root", root, "--keep-last", "1"); err != nil {
		t.Fatalf("clean: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "UTC-3")); err != nil {
		t.Fatal("newest session must be kept")
	}
	for _, id := range []string{"UTC-1", "UTC-2"} {
		if _, err := os.Stat(filepath.Join(root, id)); !os.IsNotExist(err) {
			t.Fatalf("%s should be removed by keep-last 1", id)
		}
	}
}

func TestCleanSessions_RequiresPolicy(t *testing.T) {
	if _, err := run(t, "clean", "--artifact-root", t.TempDir()); err == nil {
		t.Fatal("clean with no policy must error")
	}
}
