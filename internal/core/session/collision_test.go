package session_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func TestSessionsInOneSecondPreserveEachTestArtifact(t *testing.T) {
	base := t.TempDir()
	first, err := session.New(base, "first run", fixedTime)
	if err != nil {
		t.Fatal(err)
	}
	record := first.Test(1, "same-case")
	record.Spec([]byte(`{"id":"same-case","run":"first"}`))
	record.Status(session.StatusPass)
	if err := first.Save(); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join(first.Root(), "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := os.ReadFile(filepath.Join(record.Dir(), "spec.json"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := session.New(base, "second run", fixedTime)
	if err != nil {
		t.Fatal(err)
	}
	if first.Root() == second.Root() || first.ID() == second.ID() {
		t.Fatal("same-second runs share an artifact root")
	}
	secondRecord := second.Test(1, "same-case")
	secondRecord.Spec([]byte(`{"id":"same-case","run":"second"}`))
	secondRecord.Status(session.StatusFail)
	if err := second.Save(); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(first.Root(), "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(manifest) {
		t.Fatal("second run overwrote first manifest")
	}
	after, err = os.ReadFile(filepath.Join(record.Dir(), "spec.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(snapshot) {
		t.Fatal("second run overwrote first DSL evidence")
	}
	if filepath.Base(second.Root()) != second.ID() {
		t.Fatal("manifest ID and directory disagree")
	}
}

func TestConcurrentSameSecondSessionsHaveDistinctRoots(t *testing.T) {
	base := t.TempDir()
	var wg sync.WaitGroup
	roots := make(chan string, 16)
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s, err := session.New(base, "concurrent", fixedTime)
			if err != nil {
				t.Errorf("new: %v", err)
				return
			}
			roots <- s.Root()
		}()
	}
	wg.Wait()
	close(roots)
	seen := map[string]bool{}
	for root := range roots {
		if seen[root] {
			t.Fatalf("concurrent sessions collide: %s", root)
		}
		seen[root] = true
	}
	if len(seen) != 16 {
		t.Fatalf("got %d roots", len(seen))
	}
}
