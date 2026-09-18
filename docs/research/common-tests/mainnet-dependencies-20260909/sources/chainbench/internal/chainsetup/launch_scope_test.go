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
		"node1": {"verbosity=5"},
	}}}

	cases := map[string]struct {
		role  string
		index int
		want  string // comma-joined, in application order
	}{
		"producer node1 gets all+bp+node1": {"validator", 1, "metrics,mine,verbosity=5"},
		"producer node2 gets all+bp":       {"bp", 2, "metrics,mine"},
		"endpoint node3 gets all+en":       {"endpoint", 3, "metrics,gcmode=archive"},
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

// TestRecordLaunchSet_ValidatesScopeAndKnob refuses an unknown scope and a
// malformed knob at the point they are set, not at argv assembly.
func TestRecordLaunchSet_ValidatesScopeAndKnob(t *testing.T) {
	w := &Workspace{}
	if err := w.recordLaunchSet("pn", []string{"mine"}); err == nil {
		t.Error("an unknown scope must be refused")
	}
	if err := w.recordLaunchSet("all", []string{"=novalue"}); err == nil {
		t.Error("a malformed knob must be refused")
	}
	if err := w.recordLaunchSet("bp", []string{"mine"}); err != nil {
		t.Errorf("a role scope must be accepted: %v", err)
	}
	if err := w.recordLaunchSet("node2", []string{"verbosity=4"}); err != nil {
		t.Errorf("a node scope must be accepted: %v", err)
	}
	// Repeated records accumulate under the scope.
	_ = w.recordLaunchSet("bp", []string{"metrics"})
	if got := strings.Join(w.state.LaunchSet["bp"], ","); got != "mine,metrics" {
		t.Errorf("bp scope = %q, want mine,metrics", got)
	}
}
