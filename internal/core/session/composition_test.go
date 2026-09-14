package session_test

import (
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
