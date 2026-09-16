package session_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func compClock() time.Time { return time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC) }

type compState struct {
	Name  string                  `json:"name"`
	Steps map[string]session.Step `json:"steps"`
}

func TestCompositionRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c, err := session.OpenComposition(dir, compClock)
	if err != nil {
		t.Fatal(err)
	}

	// A never-saved composition loads the zero state (not an error).
	var st compState
	found, err := c.Load(&st)
	if err != nil {
		t.Fatalf("load of a fresh composition: %v", err)
	}
	if found {
		t.Error("a composition that was never saved must report no record")
	}
	if st.Name != "" {
		t.Fatalf("fresh state = %+v", st)
	}

	st.Name = "demo"
	st.Steps = map[string]session.Step{"new": c.StepMark("initialized")}
	if err := c.Save(st); err != nil {
		t.Fatal(err)
	}

	// A re-open sees the persisted state, with the injected clock's stamp.
	c2, err := session.OpenComposition(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	var got compState
	found, err = c2.Load(&got)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Error("a saved composition must report a record")
	}
	if got.Name != "demo" || !got.Steps["new"].Done {
		t.Fatalf("round-trip lost state: %+v", got)
	}
	if got.Steps["new"].At != "2026-08-12T09:00:00Z" {
		t.Fatalf("stamp = %q, want the injected clock", got.Steps["new"].At)
	}
}

func TestCompositionRequiresDir(t *testing.T) {
	if _, err := session.OpenComposition("", nil); err == nil {
		t.Fatal("empty dir must fail")
	}
}

// TestCompositionLoad_RefusesTheOldRecordName is why the rename looks for the
// name it replaced.
//
// The record is read only at the version this build writes; any other shape is
// refused by name. Renaming the file with no lookup would turn that refusal
// into silence: a directory holding only workspace.json reads as never
// composed, so the next command would compose a second chain beside the first
// and the operator would be told nothing.
func TestCompositionLoad_RefusesTheOldRecordName(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "workspace.json"), []byte(`{"name":"demo"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := session.OpenComposition(dir, compClock)
	if err != nil {
		t.Fatal(err)
	}

	var st compState
	found, err := c.Load(&st)
	if err == nil {
		t.Fatal("a directory holding only the old record name must be refused, not read as uncomposed")
	}
	if found {
		t.Error("a refused load must not claim it found a record")
	}
	for _, want := range []string{"workspace.json", "chain-record.json"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("refusal must name %s: %v", want, err)
		}
	}

	// The same directory with the current name loads normally, so the guard
	// keys on the old file being the ONLY one there.
	if err := c.Save(compState{Name: "demo"}); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Load(&st); err != nil {
		t.Fatalf("a directory that has both names must still load the current one: %v", err)
	}
}
