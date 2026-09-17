package chainsetup

import (
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/session"
	"slices"
	"strings"
	"testing"
	"time"
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

// TestRecordLaunchSet_HoldsOneEntryPerKey.
//
// The record answers "what was asked for", and composing the same declaration
// twice over one workspace asks for the same thing twice, not for two things.
// It used to append unconditionally, so a workspace reused across runs grew a
// duplicate line each time — harmless at argv assembly, which is
// last-write-wins, and misleading in the one place meant to say what the run
// was told to do. Measured on a workspace run three times: bp held
// ["mine=true", "mine=true", "mine=true"].
func TestRecordLaunchSet_HoldsOneEntryPerKey(t *testing.T) {
	w := &Workspace{}
	for i := 0; i < 3; i++ {
		if err := w.recordLaunchSet("bp", []string{"mine=true", "nodiscover"}); err != nil {
			t.Fatal(err)
		}
	}
	got := w.state.LaunchSet["bp"]
	if len(got) != 2 {
		t.Fatalf("three identical requests recorded %v", got)
	}

	// A new value for a key it already holds replaces that entry where it
	// stands, so the order a reader sees does not move under them.
	if err := w.recordLaunchSet("bp", []string{"mine=false"}); err != nil {
		t.Fatal(err)
	}
	got = w.state.LaunchSet["bp"]
	if len(got) != 2 || got[0] != "mine=false" || got[1] != "nodiscover" {
		t.Fatalf("after replacing mine: %v", got)
	}
}

// TestMarkStepFailed_TheRecordSaysWhereItDied.
//
// A step verb marks itself only after it succeeds, so a composition that died
// left the record showing the last step that WORKED and nothing about the one
// that did not. The reader had to know the order by heart to guess what came
// next. Measured before this: a run whose init could not exec the binary wrote
// seven "done" steps and no trace of init.
func TestMarkStepFailed_TheRecordSaysWhereItDied(t *testing.T) {
	comp, err := session.OpenComposition(t.TempDir(), func() time.Time { return time.Unix(0, 0).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	w := &Workspace{state: State{Steps: map[string]Step{}}, comp: comp}

	w.MarkStepFailed("init", errors.New(`driver: "gstable": executable file not found`))

	got, ok := w.state.Steps["init"]
	if !ok {
		t.Fatal("the failed step is not in the record")
	}
	if got.Result != session.StepFailed {
		t.Errorf("result = %q, want %q", got.Result, session.StepFailed)
	}
	if got.Done {
		t.Error("a failed step must not read as done")
	}
	if !strings.Contains(got.Err, "executable file not found") {
		t.Errorf("the record must carry why: %q", got.Err)
	}
}

// TestComposeNeeds_InitBeforeStart: the rule that a datadir is initialized
// before a node launches used to live only in the order upSteps iterates, so
// running the steps by hand (or resuming from one) could launch a node over a
// datadir no genesis had reached.
func TestComposeNeeds_InitBeforeStart(t *testing.T) {
	for step, want := range map[string]string{"init": "deploy", "start": "init"} {
		needs := composeNeeds[step]
		if len(needs) == 0 {
			t.Errorf("%s declares no prerequisite", step)
			continue
		}
		if !slices.Contains(needs, want) {
			t.Errorf("%s needs %v, want it to include %q", step, needs, want)
		}
	}
}

// TestExcerpt_KeepsBothEnds.
//
// The tail alone was kept, and the failure this evidence exists to explain
// lives at the other end: a node that refuses its genesis says so in its first
// lines and exits. At one block per second a node writes several lines a
// second, so a 200-line tail is the last half minute — the one window a startup
// failure is not in.
func TestExcerpt_KeepsBothEnds(t *testing.T) {
	lines := make([]string, 0, 1000)
	lines = append(lines, "Fatal: mismatching Boho fork block in database")
	for i := 1; i < 999; i++ {
		lines = append(lines, fmt.Sprintf("INFO imported block %d", i))
	}
	lines = append(lines, "Blockchain stopped")

	got := excerpt(strings.Join(lines, "\n"), 2, 2)

	if !strings.Contains(got, "Fatal: mismatching") {
		t.Error("the first line is where a startup failure says why; it was dropped")
	}
	if !strings.Contains(got, "Blockchain stopped") {
		t.Error("the last line was dropped")
	}
	// A reader has to know how much they are not seeing.
	if !strings.Contains(got, "996 line(s) elided") {
		t.Errorf("the excerpt must say what it left out: %q", got[:120])
	}
}

// TestExcerpt_ShortLogIsWhole: eliding nothing is not worth a marker saying so.
func TestExcerpt_ShortLogIsWhole(t *testing.T) {
	in := "one\ntwo\nthree"
	if got := excerpt(in, 2, 2); got != in {
		t.Errorf("excerpt = %q, want the whole log", got)
	}
}
