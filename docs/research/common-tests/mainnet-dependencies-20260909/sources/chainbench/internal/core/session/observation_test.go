package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// TestObservation_WritesScrubbedFile: a failed test's observation lands under
// observations/ and is scrubbed like every artifact.
func TestObservation_WritesScrubbedFile(t *testing.T) {
	sess, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	rec := sess.Test(1, "T1")
	content := []byte(`gstable --datadir /d --password s3cr3t --http`)
	rec.Observation("node1.log", content)

	path := filepath.Join(rec.Dir(), "observations", "node1.log")
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("observation not written: %v", err)
	}
	if strings.Contains(string(got), "s3cr3t") {
		t.Errorf("observation not scrubbed: %s", got)
	}
	if !strings.Contains(string(got), "--datadir /d") {
		t.Errorf("non-secret evidence lost: %s", got)
	}
}
