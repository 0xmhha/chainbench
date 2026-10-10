package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHistoryCaptureExcludesLiveDataKeysAndLinks(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "test run", time.Date(2026, 10, 8, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	r := s.Test(1, "case1")
	r.Spec([]byte(`{"version":2,"id":"case1"}`))
	r.Status(StatusPass)
	if err = s.Save(); err != nil {
		t.Fatal(err)
	}
	secret := "never-capture-this-private-file"
	for _, name := range []string{"keys/private.key", "environments/env-a/nodes/node1/secret"} {
		p := filepath.Join(s.Root(), name)
		if err = os.MkdirAll(filepath.Dir(p), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(p, []byte(secret), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = os.Symlink(filepath.Join(s.Root(), "keys/private.key"), filepath.Join(r.Dir(), "assert.json")); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(r.Dir(), "postaction.json"), []byte(`{"key":"broken\"private"`), 0600); err != nil {
		t.Fatal(err)
	}
	capture, err := Capture(root, IDFor(root, s.Root()))
	if err != nil {
		t.Fatal(err)
	}
	if capture.Result.Summary.Pass != 1 || len(capture.Gaps) != 2 {
		t.Fatal(capture)
	}
	for name, data := range capture.Files {
		if name == "tests/001_case1/postaction.json" {
			t.Fatal("malformed secret-bearing JSON captured")
		}
		if strings.Contains(data, secret) || strings.Contains(name, "keys/") || strings.Contains(name, "nodes/") {
			t.Fatalf("private/live material captured: %s", name)
		}
	}
	for _, id := range []string{"../escape", "a/b/c", `a\b`, ""} {
		if _, err = Capture(root, id); err == nil {
			t.Fatalf("unsafe reference accepted: %q", id)
		}
	}
	outside := t.TempDir()
	if err = os.Symlink(outside, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	if _, err = Capture(root, "linked"); err == nil {
		t.Fatal("linked session root accepted")
	}
}
