package collector_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestFileStore_PersistsAcrossInstances(t *testing.T) {
	path := filepath.Join(t.TempDir(), "runs.json")

	s1, err := collector.NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s1.SaveRun(collector.RunRecord{ID: "test/a", Phase: collector.PhaseTest, Chain: "wbft", Status: collector.RunSucceeded, StartedAt: time.Unix(1, 0)}); err != nil {
		t.Fatal(err)
	}
	if err := s1.SaveRun(collector.RunRecord{ID: "test/b", Phase: collector.PhaseTest, Chain: "wbft", Status: collector.RunFailed, StartedAt: time.Unix(2, 0)}); err != nil {
		t.Fatal(err)
	}

	// A fresh instance loads the persisted records.
	s2, err := collector.NewFileStore(path)
	if err != nil {
		t.Fatal(err)
	}
	runs := s2.ListRuns()
	if len(runs) != 2 || runs[0].ID != "test/a" || runs[1].Status != collector.RunFailed {
		t.Fatalf("loaded runs: %+v", runs)
	}
	if r, ok := s2.GetRun("test/b"); !ok || r.Status != collector.RunFailed {
		t.Errorf("GetRun(test/b): %+v ok=%v", r, ok)
	}
}

func TestFileStore_MissingFileIsEmpty(t *testing.T) {
	s, err := collector.NewFileStore(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if len(s.ListRuns()) != 0 {
		t.Error("expected empty store")
	}
}
