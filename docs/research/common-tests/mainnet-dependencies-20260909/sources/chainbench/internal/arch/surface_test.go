package arch

import (
	"sort"
	"testing"
)

// surfaceBudget is how many registrations may still reach past the app layer,
// per surface. It is a ceiling that only comes down, and both surfaces are at
// zero as of 2026-09-05 (U6).
//
// The DSL is deliberately absent, and that is a correction rather than an
// omission. The rule as first written said "CLI, MCP and DSL all reach a
// feature through app", but the DSL's actions are not a surface: they are the
// implementation of the language's vocabulary, at L3 (layers.md §3,
// `testhelper`), which is BELOW app. Making them call app is not merely a layer
// violation, it is an import cycle — app imports testengine, which reaches
// testhelper — and the compiler says so.
//
// What is a surface for the DSL is `run`, and both spellings of it (the CLI
// command and chainbench_run) already go through app. Counting the vocabulary
// as 45 registrations in debt made the number unreachable by construction, and
// a ceiling that cannot come down teaches the next reader to ignore it.
//
// Lower a number when its surface's entries move under app. Never raise one. A
// surface that comes in under budget fails too, so the ceiling tracks reality
// rather than drifting above it.
var surfaceBudget = map[string]int{
	"CLI": 0,
	"MCP": 0,
}

// vocabulary names the entry kinds that are a language's implementation rather
// than a surface: the DSL's actions and assertions, which live at L3 and cannot
// reach up to app. They are inventoried, not budgeted.
var vocabulary = map[string]bool{"DSL": true, "DSLa": true}

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
		if _, ok := surfaceBudget[s]; !ok && !vocabulary[s] {
			t.Errorf("surface %q has no budget, so nothing holds it down; add it to surfaceBudget", s)
		}
	}

	if t.Failed() {
		return
	}
	pastSurfaces, vocab := 0, 0
	for s, n := range past {
		if vocabulary[s] {
			vocab += n
			continue
		}
		pastSurfaces += n
	}
	t.Logf("%d of %d surface registrations reach past app; the DSL vocabulary is %d entries at L3, which is where it belongs",
		pastSurfaces, len(entries)-vocab, vocab)
}
