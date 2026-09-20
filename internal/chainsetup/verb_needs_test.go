package chainsetup

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestEveryVerbDeclaresWhatItNeeds is the ratchet the declaration exists for.
//
// Before it, a verb added to Workspace checked whatever its author remembered
// and said so in whatever words they chose. Four verbs ended up refusing an
// empty node table in four wordings, sending a reader to three different places
// for one missing thing. A table nobody is obliged to fill would have drifted
// the same way, so the obligation is here: the method set is walked, and a verb
// that is not declared fails the build.
//
// "Needs nothing" stays a legal answer — an accessor really does need nothing —
// but it has to be written down with its reason, so the next reader can tell a
// decision from an oversight.
func TestEveryVerbDeclaresWhatItNeeds(t *testing.T) {
	typ := reflect.TypeOf(&Workspace{})
	var undeclared, unexplained []string
	verbs := 0
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		verbs++
		need, ok := verbNeeds[name]
		if !ok {
			undeclared = append(undeclared, name)
			continue
		}
		if need.step == "" && need.run == anyRun && need.why == "" {
			unexplained = append(unexplained, name)
		}
	}
	if verbs == 0 {
		t.Fatal("no exported methods found, so the walk is wrong rather than the surface empty")
	}
	sort.Strings(undeclared)
	sort.Strings(unexplained)
	if len(undeclared) > 0 {
		t.Errorf("these verbs are not in verbNeeds, so nothing says what they require:\n  %s\n"+
			"Declare a step, a run state, or why they need neither.", strings.Join(undeclared, "\n  "))
	}
	if len(unexplained) > 0 {
		t.Errorf("these verbs declare no requirement and no reason:\n  %s\n"+
			"Write why. \"It checks nothing\" has to be a decision, not a gap.", strings.Join(unexplained, "\n  "))
	}
	t.Logf("%d verbs declared", verbs)
}

// TestVerbNeedsNamesNoGhost is the reverse: an entry for a verb that no longer
// exists means the table is describing a surface that is gone, which is how the
// worklist and the layer table each rotted before a check was put on them.
func TestVerbNeedsNamesNoGhost(t *testing.T) {
	typ := reflect.TypeOf(&Workspace{})
	have := map[string]bool{}
	for i := 0; i < typ.NumMethod(); i++ {
		have[typ.Method(i).Name] = true
	}
	var ghosts []string
	for name := range verbNeeds {
		if !have[name] {
			ghosts = append(ghosts, name)
		}
	}
	sort.Strings(ghosts)
	if len(ghosts) > 0 {
		t.Errorf("verbNeeds names verbs that do not exist:\n  %s", strings.Join(ghosts, "\n  "))
	}
}

// TestVerbNeedsStepsAreRealSteps holds the step half to composeNeeds. A verb
// naming a step the composition does not know would pass require() silently —
// the map returns a nil slice for an unknown key, so every prerequisite of a
// typo is vacuously satisfied.
func TestVerbNeedsStepsAreRealSteps(t *testing.T) {
	for verb, need := range verbNeeds {
		if need.step == "" {
			continue
		}
		if _, ok := composeNeeds[need.step]; !ok {
			t.Errorf("%s declares step %q, which composeNeeds does not know — require() would pass it silently", verb, need.step)
		}
	}
}

// TestAllowRefusesAnEmptyTable checks the message a person actually meets, and
// that the run state is read rather than assumed.
func TestAllowRefusesAnEmptyTable(t *testing.T) {
	w := &Workspace{}
	err := w.allow("Health")
	if err == nil {
		t.Fatal("an empty node table passed a verb that needs one placed")
	}
	if !strings.Contains(err.Error(), "node table is empty") || !strings.Contains(err.Error(), "chain place") {
		t.Errorf("the refusal does not say what is missing or what to run: %v", err)
	}
	w.state.Nodes = []node.Record{{Index: 1}}
	if err := w.allow("Health"); err != nil {
		t.Errorf("a placed table was refused: %v", err)
	}
}

// TestAllowRefusesRemovingALiveNode is the destructive half.
func TestAllowRefusesRemovingALiveNode(t *testing.T) {
	w := &Workspace{}
	w.state.Nodes = []node.Record{{Index: 1}, {Index: 2, PID: 4242}}
	err := w.allow("Rm")
	if err == nil {
		t.Fatal("rm was allowed over a running node")
	}
	if !strings.Contains(err.Error(), "node2") || !strings.Contains(err.Error(), "4242") {
		t.Errorf("the refusal does not name the node holding it up: %v", err)
	}
	w.state.Nodes[1].PID = 0
	if err := w.allow("Rm"); err != nil {
		t.Errorf("a stopped network was refused: %v", err)
	}
}

// TestAllowRefusesAnUndeclaredVerb keeps allow() honest for a caller that
// arrives before the table does.
func TestAllowRefusesAnUndeclaredVerb(t *testing.T) {
	w := &Workspace{}
	if err := w.allow("NotAVerb"); err == nil {
		t.Fatal("an undeclared verb was allowed")
	}
}
