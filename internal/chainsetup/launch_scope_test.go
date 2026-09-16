package chainsetup

import (
	"strings"
	"testing"
)

// TestLaunchOverridesFor_MergesScopesMostGeneralFirst locks the per-node scope
// fold: "all" applies to every node, a role scope to that role, "node<N>" to the
// one node, and the node's own scope comes last so it wins at assembly.
func TestLaunchOverridesFor_MergesScopesMostGeneralFirst(t *testing.T) {
	w := &Workspace{state: State{LaunchSet: map[string][]string{
		"all":   {"metrics"},
		"bp":    {"mine"},
		"en":    {"gcmode=archive"},
		"pn":    {"maxpeers=200"},
		"node1": {"verbosity=5"},
	}}}

	cases := map[string]struct {
		role  string
		index int
		want  string // comma-joined, in application order
	}{
		"producer node1 gets all+bp+node1": {"bp", 1, "metrics,mine,verbosity=5"},
		"producer node2 gets all+bp":       {"bp", 2, "metrics,mine"},
		"endpoint node3 gets all+en":       {"en", 3, "metrics,gcmode=archive"},
		"proxy node4 gets all+pn":          {"pn", 4, "metrics,maxpeers=200"},
		"an unreadable role gets all only": {"sideways", 5, "metrics"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := strings.Join(w.launchOverridesFor(tc.role, tc.index), ",")
			if got != tc.want {
				t.Errorf("launchOverridesFor(%q,%d) = %q, want %q", tc.role, tc.index, got, tc.want)
			}
		})
	}
}

// TestRecordLaunchSet_AcceptsEveryRoleAndRefusesWhatIsNotAScope is the
// regression for a proxy tier that could be declared but not configured.
//
// The scope list used to be written out by hand here and again in the grammar,
// and both lists named bp and en and forgot pn. A topology could declare a pn,
// the composer would launch it, and a launch flag scoped to "pn" was refused as
// an unknown scope — so the tier existed but nothing could tune it. The list is
// the vocabulary's now, and a role added there is addressable here without a
// second edit.
func TestRecordLaunchSet_AcceptsEveryRoleAndRefusesWhatIsNotAScope(t *testing.T) {
	for _, scope := range []string{"all", "bp", "en", "pn", "node2", "node12"} {
		w := &Workspace{}
		if err := w.recordLaunchSet(scope, []string{"mine"}); err != nil {
			t.Errorf("scope %q must be accepted: %v", scope, err)
		}
	}
	for _, scope := range []string{"validator", "endpoint", "boot", "sideways", "node0", "node", ""} {
		w := &Workspace{}
		if err := w.recordLaunchSet(scope, []string{"mine"}); err == nil {
			t.Errorf("scope %q must be refused", scope)
		}
	}
}

// TestRecordLaunchSet_ValidatesTheKnobAndAccumulates: a malformed override is
// refused where it is set rather than at argv assembly, and repeated records
// under one scope accumulate in order.
func TestRecordLaunchSet_ValidatesTheKnobAndAccumulates(t *testing.T) {
	w := &Workspace{}
	if err := w.recordLaunchSet("all", []string{"=novalue"}); err == nil {
		t.Error("a malformed knob must be refused")
	}
	if err := w.recordLaunchSet("bp", []string{"mine"}); err != nil {
		t.Fatalf("a role scope must be accepted: %v", err)
	}
	if err := w.recordLaunchSet("bp", []string{"metrics"}); err != nil {
		t.Fatalf("a repeated record must be accepted: %v", err)
	}
	if got := strings.Join(w.state.LaunchSet["bp"], ","); got != "mine,metrics" {
		t.Errorf("bp scope = %q, want mine,metrics", got)
	}
}
