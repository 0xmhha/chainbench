package testengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSaveFailureData_LeavesEvidenceWhereThereIsNoSession is the point of
// splitting gathering from writing.
//
// A run that fails before its first test has no session and no test record, so
// the evidence used to have nowhere to go and was dropped. It reported "1 node
// still not ready" and threw away the only thing that could say which node.
func TestSaveFailureData_LeavesEvidenceWhereThereIsNoSession(t *testing.T) {
	dir := t.TempDir()
	at := time.Date(2026, 9, 17, 4, 24, 48, 0, time.UTC)

	got, err := saveFailureData(func() time.Time { return at }, dir, []evidence{
		{Name: "health.json", Data: []byte(`{"producing":false}`)},
		{Name: "node1.log", Data: []byte("Fatal: something\n")},
	})
	if err != nil {
		t.Fatal(err)
	}
	// The directory is stamped so two failed attempts on one workspace do not
	// overwrite each other.
	if want := filepath.Join(dir, failureDir, "20260917-042448"); got != want {
		t.Fatalf("evidence went to %q, want %q", got, want)
	}
	for _, name := range []string{"health.json", "node1.log"} {
		if _, err := os.Stat(filepath.Join(got, name)); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestSaveFailureData_ScrubsSecrets: the evidence holds a node's launch command,
// and that command carries a password path and an unlock address. It is written
// through the same scrubber a test record uses, so the two paths cannot disagree
// about what counts as a secret.
func TestSaveFailureData_ScrubsSecrets(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`gstable --password /k/password --unlock 0xc17d493883eaa3b4cceb0f214b273392d562f9d8`)

	got, err := saveFailureData(nil, dir, []evidence{{Name: "processes.json", Data: raw}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(got, "processes.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "/k/password") {
		t.Errorf("the password path reached the evidence: %s", b)
	}
}

// TestSaveFailureData_NothingGatheredWritesNothing: an empty gather is not a
// failure to report. A workspace that never composed has nothing to say, and a
// directory holding no files would only make an operator look for one.
func TestSaveFailureData_NothingGatheredWritesNothing(t *testing.T) {
	dir := t.TempDir()
	got, err := saveFailureData(nil, dir, nil)
	if err != nil || got != "" {
		t.Fatalf("saveFailureData(nil) = %q, %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, failureDir)); !os.IsNotExist(err) {
		t.Errorf("an empty gather created %s", failureDir)
	}
}
