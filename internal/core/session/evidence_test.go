package session

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func evidenceSession(t *testing.T) (root, ref string, failed TestRecord) {
	t.Helper()
	root = t.TempDir()
	s, err := New(root, "test run", time.Date(2026, 10, 27, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	passed := s.Test(1, "passes")
	passed.Status(StatusPass)
	failed = s.Test(2, "fails")
	failed.Status(StatusFail)
	failed.Observation("health.json", []byte(`{"nodes":[]}`))
	failed.Observation("node1.log", []byte("INFO started\nWARN slow\n"))
	failed.Observation("gather-problems.txt", []byte("logs: node2 unreachable\n"))
	if err = s.Save(); err != nil {
		t.Fatal(err)
	}
	return root, IDFor(root, s.Root()), failed
}

// TestEvidenceListsWhatFailedTestsLeft: only tests with gathered evidence are
// listed, each file with its size.
func TestEvidenceListsWhatFailedTestsLeft(t *testing.T) {
	root, ref, _ := evidenceSession(t)
	got, err := ListEvidence(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	want := []EvidenceTest{{Dir: "tests/002_fails", Files: []EvidenceFile{{Name: "gather-problems.txt", Bytes: 24}, {Name: "health.json", Bytes: 12}, {Name: "node1.log", Bytes: 23}}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("evidence = %+v\nwant       %+v", got, want)
	}
	content, err := ReadEvidence(root, ref, "tests/002_fails", "node1.log")
	if err != nil || string(content) != "INFO started\nWARN slow\n" {
		t.Fatalf("read = %q %v", content, err)
	}
}

// TestEvidenceReadsNothingElse: a reference, test or file name outside the
// evidence set is refused, and so is a link, however it is spelled.
func TestEvidenceReadsNothingElse(t *testing.T) {
	root, ref, failed := evidenceSession(t)
	secret := filepath.Join(filepath.Dir(filepath.Dir(failed.Dir())), "keys", "private.key")
	if err := os.MkdirAll(filepath.Dir(secret), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(secret, []byte("private"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(secret, filepath.Join(failed.Dir(), "observations", "node2.log")); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ ref, test, name string }{
		{ref, "tests/002_fails", "../../keys/private.key"},
		{ref, "tests/002_fails/../../keys", "private.key"},
		{ref, "keys", "private.key"},
		{ref, "tests/002_fails", "status.json"},
		{ref, "tests/002_fails", "node2.log"},
		{"../" + ref, "tests/002_fails", "node1.log"},
	} {
		if content, err := ReadEvidence(root, c.ref, c.test, c.name); err == nil {
			t.Errorf("read %+v = %q, want a refusal", c, content)
		}
	}
	listed, err := ListEvidence(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range listed[0].Files {
		if f.Name == "node2.log" {
			t.Fatal("a linked file was listed as evidence")
		}
	}
}

// TestEvidenceFollowsNoLinkInsideTheSession: an observations directory that
// is a link, or an evidence file hard-linked to another file, is neither
// listed nor read, even though it stays inside the session.
func TestEvidenceFollowsNoLinkInsideTheSession(t *testing.T) {
	root, ref, failed := evidenceSession(t)
	sessionRoot := filepath.Dir(filepath.Dir(failed.Dir()))
	other := filepath.Join(sessionRoot, "environments", "env-x")
	if err := os.MkdirAll(other, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "node1.log"), []byte("unscrubbed"), 0o600); err != nil {
		t.Fatal(err)
	}
	// A second failed test whose observations directory is a link.
	linked := filepath.Join(sessionRoot, "tests", "003_linked")
	if err := os.MkdirAll(linked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(other, filepath.Join(linked, "observations")); err != nil {
		t.Fatal(err)
	}
	// A hard link planted beside real evidence.
	if err := os.Link(filepath.Join(other, "node1.log"), filepath.Join(failed.Dir(), "observations", "node9.log")); err != nil {
		t.Fatal(err)
	}
	listed, err := ListEvidence(root, ref)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range listed {
		if test.Dir == "tests/003_linked" {
			t.Fatal("a linked observations directory was listed")
		}
		for _, f := range test.Files {
			if f.Name == "node9.log" {
				t.Fatal("a hard-linked file was listed")
			}
		}
	}
	for _, c := range [][2]string{{"tests/003_linked", "node1.log"}, {"tests/002_fails", "node9.log"}} {
		if content, err := ReadEvidence(root, ref, c[0], c[1]); err == nil {
			t.Errorf("read %v = %q, want a refusal", c, content)
		}
	}
}

// TestEvidenceRefusesALinkedTestsDirectory: the tests directory itself must be
// a real directory, or a link could serve files the engine never scrubbed.
func TestEvidenceRefusesALinkedTestsDirectory(t *testing.T) {
	root, ref, failed := evidenceSession(t)
	sessionRoot := filepath.Dir(filepath.Dir(failed.Dir()))
	moved := filepath.Join(sessionRoot, "environments", "moved")
	if err := os.MkdirAll(filepath.Dir(moved), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(sessionRoot, "tests"), moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("environments/moved", filepath.Join(sessionRoot, "tests")); err != nil {
		t.Fatal(err)
	}
	if listed, err := ListEvidence(root, ref); err == nil && len(listed) != 0 {
		t.Fatalf("a linked tests directory was listed: %+v", listed)
	}
	if content, err := ReadEvidence(root, ref, "tests/002_fails", "node1.log"); err == nil {
		t.Fatalf("read through a linked tests directory = %q", content)
	}
}
