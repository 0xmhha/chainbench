package testengine

import (
	"context"
	"strings"
	"testing"
)

// planFor composes the plan the way RunSuite does: through compositionOf, so a
// test cannot assert on a value the runner would never see.
func planFor(t *testing.T, env string, in RunSuiteIn) ComposePlan {
	t.Helper()
	spec := caseWithEnv(t, env)
	in.DataDir = t.TempDir()
	comp, err := compositionOf(context.Background(), spec, in)
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	return planOf(comp, spec.Chain.Name)
}

// TestPlan_CountsRolesOffANodeTable is the mapping most likely to be wrong and
// least likely to be noticed: with a node table the composition carries no
// counts at all (BPCount and friends stay zero), so a plan that read them would
// print "bp 0 · en 0 · pn 0" over a fifteen node network and look plausible.
func TestPlan_CountsRolesOffANodeTable(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},
	  "topology":{"nodes":[
	    {"index":1,"role":"bp"},{"index":2,"role":"bp"},
	    {"index":3,"role":"en"},{"index":4,"role":"pn"}]}}`
	p := planFor(t, env, RunSuiteIn{})

	if p.Nodes.BP != 2 || p.Nodes.EN != 1 || p.Nodes.PN != 1 {
		t.Fatalf("roles = bp %d en %d pn %d, want 2/1/1", p.Nodes.BP, p.Nodes.EN, p.Nodes.PN)
	}
	if !p.Nodes.Declared {
		t.Error("a node table must be reported as declared, so a reader knows the counts were read and not requested")
	}
	if got := p.Nodes.line(); !strings.Contains(got, "bp 2 · en 1 · pn 1") {
		t.Errorf("nodes line = %q", got)
	}
}

// TestPlan_CountsComeFromTheRequestWithoutATable is the other form: counts, and
// an explicit --bp overriding the declaration.
func TestPlan_CountsComeFromTheRequestWithoutATable(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":3,"en":2,"syncMode":"snap"}}`

	p := planFor(t, env, RunSuiteIn{})
	if p.Nodes.BP != 3 || p.Nodes.EN != 2 || p.Nodes.Declared {
		t.Fatalf("declared counts = %+v", p.Nodes)
	}
	if p.Nodes.SyncMode != "snap" {
		t.Errorf("sync mode = %q, want snap", p.Nodes.SyncMode)
	}

	over := planFor(t, env, RunSuiteIn{BPCount: 7})
	if over.Nodes.BP != 7 {
		t.Fatalf("--bp 7 did not reach the plan: bp = %d", over.Nodes.BP)
	}
}

// TestPlan_ShowsBothOverrideLayers is the reason the plan exists. The
// declaration's scoped launch and the command's --launch-opt arrive on two
// different fields and are applied at two different precedences; a reader who
// cannot see both has to merge them in their head, which is the thing that goes
// wrong.
func TestPlan_ShowsBothOverrideLayers(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":2},
	  "launch":{"bp":{"mine":true}},
	  "config":{"node1":{"txpool.pricelimit":1}}}`
	p := planFor(t, env, RunSuiteIn{LaunchOpts: []string{"nodiscover"}, NetworkID: 4242})

	if got := p.Launch["bp"]; len(got) != 1 || !strings.HasPrefix(got[0], "mine") {
		t.Errorf("declared bp launch = %v", got)
	}
	all := strings.Join(p.Launch["all"], " ")
	if !strings.Contains(all, "nodiscover") || !strings.Contains(all, "4242") {
		t.Errorf("the command's opts must land in the all scope: %q", all)
	}
	if got := p.Config["node1"]; len(got) != 1 {
		t.Errorf("declared config = %v", got)
	}

	// Rendered most-general-first, so the reader sees them in the order they
	// will be applied.
	rendered := p.String()
	allAt, bpAt := strings.Index(rendered, "all:"), strings.Index(rendered, "bp:")
	if allAt < 0 || bpAt < 0 || allAt > bpAt {
		t.Errorf("scopes out of order:\n%s", rendered)
	}
}

// TestPlan_KeysSourceDefaultsToTheRecordedSet: a declaration that names no
// source composes from the recorded key set, and the plan has to say so rather
// than print an empty field, because "" reads as "nothing decided yet".
func TestPlan_KeysSourceDefaultsToTheRecordedSet(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":2}}`
	p := planFor(t, env, RunSuiteIn{})
	if p.Keys.Source != keySourceKeyPreset {
		t.Fatalf("keys source = %q, want %q", p.Keys.Source, keySourceKeyPreset)
	}
	if p.Keys.Dir != defaultKeysDir {
		t.Errorf("keys dir = %q, want %q", p.Keys.Dir, defaultKeysDir)
	}
}

// TestPlan_HandoffSaysWhichBinaryTakesOver: an upgrade env composes no layout,
// so the plan answers a different set of questions. It must not print a network
// of zero nodes.
func TestPlan_HandoffSaysWhichBinaryTakesOver(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
	  "binaries":{"from":"gwemix","to":"gwbft"},
	  "upgrade":{"profile":"profiles/p.yaml","template":"t.json"}}`
	p := planFor(t, env, RunSuiteIn{})

	if p.Handoff == nil {
		t.Fatal("an upgrade env must plan as a handoff")
	}
	if p.Handoff.FromBinary != "gwemix" || p.Handoff.ToBinary != "gwbft" {
		t.Fatalf("binaries = %+v", p.Handoff)
	}
	rendered := p.String()
	if strings.Contains(rendered, "bp 0") {
		t.Errorf("a handoff plan must not print a layout it does not have:\n%s", rendered)
	}
	if !strings.Contains(rendered, "from gwemix -> to gwbft") {
		t.Errorf("rendering:\n%s", rendered)
	}
}

// TestPlan_TargetNamesWhereTheNodesRun: the same specs compose a different
// network on a server set, and that is exactly the difference an operator wants
// confirmed before a remote run.
func TestPlan_TargetNamesWhereTheNodesRun(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":2}}`
	if got := planFor(t, env, RunSuiteIn{}).Target; got != "this machine" {
		t.Errorf("target = %q", got)
	}
	in := RunSuiteIn{Docker: true}
	in.Server.Name = "server-01"
	if got := planFor(t, env, in).Target; got != "server server-01 (docker)" {
		t.Errorf("target = %q", got)
	}
}
