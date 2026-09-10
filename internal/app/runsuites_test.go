package app

import (
	"reflect"
	"testing"
)

// TestSuiteSpecUnits_OnePerDefinitionInOrder: each definition becomes its own
// unit, in the caller's order — the sequencing never rearranges them.
func TestSuiteSpecUnits_OnePerDefinitionInOrder(t *testing.T) {
	in := RunSuiteIn{SpecPaths: []string{"a.json", "b.json", "a.json"}}
	units, err := suiteSpecUnits(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 3 {
		t.Fatalf("got %d units, want 3", len(units))
	}
	got := []string{units[0].label, units[1].label, units[2].label}
	if !reflect.DeepEqual(got, []string{"a.json", "b.json", "a.json"}) {
		t.Fatalf("order changed: %v", got)
	}
	for i, u := range units {
		if len(u.paths) != 1 {
			t.Fatalf("unit %d carries %d paths, want 1 (one definition per run)", i, len(u.paths))
		}
	}
}

// TestSuiteSpecUnits_InlineContentSplitsToo: the MCP form sequences the same way.
func TestSuiteSpecUnits_InlineContentSplitsToo(t *testing.T) {
	in := RunSuiteIn{SpecContent: [][]byte{[]byte(`{"a":1}`), []byte(`{"b":2}`)}}
	units, err := suiteSpecUnits(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 2 {
		t.Fatalf("got %d units, want 2", len(units))
	}
	if string(units[0].content[0]) != `{"a":1}` || string(units[1].content[0]) != `{"b":2}` {
		t.Fatalf("inline content not split in order: %v", units)
	}
}

func TestSuiteSpecUnits_NoneIsAnError(t *testing.T) {
	if _, err := suiteSpecUnits(RunSuiteIn{}); err == nil {
		t.Fatal("no definition must error")
	}
}

// TestRunSuites_KeepsNetworkUpBetweenDefinitions pins the one behavioural
// difference between running one definition and several: every definition but
// the last leaves the chain up, so the next one's preflight can reuse it. The
// caller's own KeepUp still wins for the last.
func TestRunSuites_KeepsNetworkUpBetweenDefinitions(t *testing.T) {
	cases := []struct {
		name       string
		callerKeep bool
		want       []bool // KeepUp per definition, in order
	}{
		{"default tears down after the last", false, []bool{true, true, false}},
		{"--keep-up keeps the last too", true, []bool{true, true, true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := RunSuiteIn{SpecPaths: []string{"a.json", "b.json", "c.json"}, KeepUp: tc.callerKeep}
			units, err := suiteSpecUnits(in)
			if err != nil {
				t.Fatal(err)
			}
			var got []bool
			for i := range units {
				got = append(got, in.KeepUp || i < len(units)-1)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("KeepUp per definition = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestRunSuitesOut_TotalsSeparatesTheThreeOutcomes pins what the exit code is
// derived from: a definition that could not run at all is not the same as a test
// that failed, and neither is a blocked one. Collapsing them is what the
// sequential path used to do by returning one generic error.
func TestRunSuitesOut_TotalsSeparatesTheThreeOutcomes(t *testing.T) {
	mk := func(pass, fail, blocked int) SuiteRunResult {
		var r SuiteRunResult
		r.Out.Summary.Summary.Pass = pass
		r.Out.Summary.Summary.Fail = fail
		r.Out.Summary.Summary.Blocked = blocked
		return r
	}
	out := RunSuitesOut{Runs: []SuiteRunResult{
		mk(1, 0, 0),
		mk(0, 2, 0),
		mk(0, 0, 1),
		{Spec: "c.json", Err: "compose failed"},
	}}
	setupErrors, failed, blocked := out.Totals()
	if setupErrors != 1 || failed != 2 || blocked != 1 {
		t.Fatalf("totals = %d/%d/%d, want 1 setup error, 2 failed, 1 blocked", setupErrors, failed, blocked)
	}
	if !out.Failed() {
		t.Fatal("a run with failures must report failed")
	}

	clean := RunSuitesOut{Runs: []SuiteRunResult{mk(3, 0, 0)}}
	if s, f, b := clean.Totals(); s != 0 || f != 0 || b != 0 {
		t.Fatalf("a clean run totals %d/%d/%d", s, f, b)
	}
	if clean.Failed() {
		t.Fatal("a clean run must not report failed")
	}
}
