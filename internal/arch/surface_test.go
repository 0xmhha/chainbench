package arch

import (
	"fmt"
	"sort"
	"testing"
)

// surfaceBudget is how many registrations may still reach past the app layer,
// per surface. It is a ceiling that only comes down.
//
// The rule is that CLI, MCP and DSL all reach a feature through app
// (architecture-v2 §2, decided 2026-09-05). 109 registrations do not yet, and
// moving them is the U track (worklist §1l). Until that finishes the rule
// cannot be asserted outright, so it is held as a budget instead: new work may
// not add to the debt, and each U item lowers these numbers.
//
// Lower a number when its surface's entries move under app. Never raise one. A
// surface that comes in under budget fails too, so the ceiling tracks reality
// rather than drifting above it.
var surfaceBudget = map[string]int{
	"CLI":  40,
	"MCP":  24,
	"DSL":  18,
	"DSLa": 27,
}

// TestSurfacesReachThroughApp holds the U track's ratchet.
//
// It counts rather than forbids because the alternative was tried: the old rule
// was an import allowlist on one surface, and it did not keep the surfaces
// together. What the rule is for is that two surfaces answer the same question
// alike, and that is proven per feature by an equivalence test — see
// cmd/chainbench/resourcecmd/parity_test.go for the shape. This test only makes
// sure the pile of features still needing one keeps shrinking.
func TestSurfacesReachThroughApp(t *testing.T) {
	entries := Entries("../..")
	if len(entries) == 0 {
		t.Fatal("no surface registrations were found, so this test proves nothing — the walk is broken")
	}

	past := map[string]int{}
	examples := map[string][]string{}
	for _, e := range entries {
		if !e.ReachesPastApp() {
			continue
		}
		past[e.Surface]++
		examples[e.Surface] = append(examples[e.Surface], e.Name)
	}

	surfaces := make([]string, 0, len(surfaceBudget))
	for s := range surfaceBudget {
		surfaces = append(surfaces, s)
	}
	sort.Strings(surfaces)

	for _, s := range surfaces {
		budget, got := surfaceBudget[s], past[s]
		switch {
		case got > budget:
			sort.Strings(examples[s])
			t.Errorf("%s: %d registrations reach past app, over the budget of %d.\n"+
				"  A surface reaches a feature through app (architecture-v2 §2); add an app entry point\n"+
				"  and call it, rather than the module, from the surface.\n"+
				"  current: %v", s, got, budget, examples[s])
		case got < budget:
			t.Errorf("%s: only %d registrations reach past app, under the budget of %d.\n"+
				"  Work landed without lowering the ceiling. Set surfaceBudget[%q] = %d.",
				s, got, budget, s, got)
		}
	}

	for s := range past {
		if _, ok := surfaceBudget[s]; !ok {
			t.Errorf("surface %q has no budget, so nothing holds it down; add it to surfaceBudget", s)
		}
	}

	if t.Failed() {
		return
	}
	total := 0
	for _, n := range past {
		total += n
	}
	t.Log(remaining(past, total, len(entries)))
}

// remaining renders the countdown, so a passing run still says how far the
// track has to go.
func remaining(past map[string]int, total, registrations int) string {
	s := fmt.Sprintf("%d of %d registrations still reach past app", total, registrations)
	keys := make([]string, 0, len(past))
	for k := range past {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		s += fmt.Sprintf("; %s %d", k, past[k])
	}
	return s
}
