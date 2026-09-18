package chainsetup

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestComposeNeeds_EveryStepRefusesEachMissingPrerequisite is N9: the
// resolution order is declared once and every step is held to it.
//
// It walks the declaration rather than listing the pairs, so a dependency added
// to composeNeeds is covered the moment it is written down. That matters more
// than it looks: the checks this replaced were per-step and disagreed with each
// other, and nothing noticed because no test knew what the order was supposed
// to be.
func TestComposeNeeds_EveryStepRefusesEachMissingPrerequisite(t *testing.T) {
	for step, needs := range composeNeeds {
		for _, missing := range needs {
			w := &Workspace{}
			w.state.Steps = map[string]Step{}
			for _, done := range needs {
				if done != missing {
					w.state.Steps[done] = Step{}
				}
			}
			err := w.require(step)
			if err == nil {
				t.Errorf("%s ran with %s missing", step, missing)
				continue
			}
			// The message has to name the step the operator should run, not
			// just say something is absent. A composition is resumed by hand.
			if !strings.Contains(err.Error(), missing) {
				t.Errorf("%s with %s missing said %q, which does not name %s", step, missing, err, missing)
			}
		}
		// With everything present the step is free to run.
		w := &Workspace{}
		w.state.Steps = map[string]Step{}
		for _, done := range needs {
			w.state.Steps[done] = Step{}
		}
		if err := w.require(step); err != nil {
			t.Errorf("%s refused with every prerequisite present: %v", step, err)
		}
	}
}

// TestComposeNeeds_IsAcyclicAndReachable keeps the declaration from describing
// an order that cannot be walked. A cycle would deadlock a composition with an
// error message on both ends, and a step needing something no step produces
// would never run at all.
func TestComposeNeeds_IsAcyclicAndReachable(t *testing.T) {
	produced := map[string]bool{"new": true}
	for step := range composeNeeds {
		produced[step] = true
	}
	for step, needs := range composeNeeds {
		for _, n := range needs {
			if !produced[n] {
				t.Errorf("%s needs %q, which no step marks", step, n)
			}
		}
	}

	var walk func(step string, seen map[string]bool)
	walk = func(step string, seen map[string]bool) {
		if seen[step] {
			t.Fatalf("composeNeeds has a cycle through %q", step)
		}
		seen[step] = true
		for _, n := range composeNeeds[step] {
			next := map[string]bool{}
			for k := range seen {
				next[k] = true
			}
			walk(n, next)
		}
	}
	for step := range composeNeeds {
		walk(step, map[string]bool{})
	}
}

// TestGenesis_RefusesBeforePlace is the same rule reached through the use case,
// so the guard is proven to be wired in and not merely present.
func TestGenesis_RefusesBeforePlace(t *testing.T) {
	dir := t.TempDir()
	d := Deps{Clock: func() time.Time { return time.Unix(0, 0).UTC() }}
	if _, err := NetNew(context.Background(), d, NetNewIn{DataDir: dir, Chain: "wbft"}); err != nil {
		t.Fatalf("new: %v", err)
	}
	_, err := NetGenesis(context.Background(), d, NetGenesisIn{DataDir: dir})
	if err == nil {
		t.Fatal("genesis composed with no placement")
	}
	if !strings.Contains(err.Error(), "place") {
		t.Errorf("error %q should tell the operator to run place", err)
	}
}
