package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// A failed test is read against the node logs of the moments it ran, so its
// record keeps when it started and when its verdict was written.
func TestTestRecordKeepsWhenItStartedAndFinished(t *testing.T) {
	start := time.Date(2026, 10, 23, 7, 0, 0, 0, time.UTC)
	s, err := New(t.TempDir(), "chainbench run", start)
	if err != nil {
		t.Fatal(err)
	}
	clock := start.Add(5 * time.Second)
	s.(*sess).now = func() time.Time { return clock }

	rec := s.Test(1, "transfer")
	clock = clock.Add(42 * time.Second)
	rec.Status(StatusFail)
	clock = clock.Add(time.Second)
	rec.Reason("balance mismatch") // A later reason does not move the finish.

	raw, err := os.ReadFile(filepath.Join(rec.Dir(), fileStatus))
	if err != nil {
		t.Fatal(err)
	}
	var status struct {
		StartedAt, FinishedAt string
	}
	if err = json.Unmarshal(raw, &status); err != nil {
		t.Fatal(err)
	}
	if status.StartedAt != "2026-10-23T07:00:05Z" || status.FinishedAt != "2026-10-23T07:00:47Z" {
		t.Fatalf("status.json times = %s", raw)
	}

	if err = s.Save(); err != nil {
		t.Fatal(err)
	}
	result, err := LoadDir(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tests) != 1 || result.Tests[0].StartedAt != status.StartedAt || result.Tests[0].FinishedAt != status.FinishedAt {
		t.Fatalf("session.json tests = %+v", result.Tests)
	}
}

// A test that never got a verdict has a start but no finish; the reader must
// not invent one.
func TestUnfinishedTestRecordHasNoFinish(t *testing.T) {
	start := time.Date(2026, 10, 23, 7, 0, 0, 0, time.UTC)
	s, err := New(t.TempDir(), "chainbench run", start)
	if err != nil {
		t.Fatal(err)
	}
	s.(*sess).now = func() time.Time { return start }
	s.Test(1, "unfinished")
	if err = s.Save(); err != nil {
		t.Fatal(err)
	}
	result, err := LoadDir(s.Root())
	if err != nil {
		t.Fatal(err)
	}
	if got := result.Tests[0]; got.StartedAt == "" || got.FinishedAt != "" {
		t.Fatalf("unfinished test = %+v", got)
	}
}
