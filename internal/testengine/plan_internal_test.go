package testengine

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
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

	if got := p.Launch["bp"]; len(got) != 1 || !strings.HasPrefix(got[0].Knob, "mine") {
		t.Errorf("declared bp launch = %v", got)
	} else if got[0].From != SourceDeclaration {
		t.Errorf("the bp knob came from the declaration, not %q", got[0].From)
	}
	var all []string
	for _, k := range p.Launch["all"] {
		all = append(all, k.Knob)
		if k.From != SourceCommand {
			t.Errorf("%q landed in the all scope from the command, not %q", k.Knob, k.From)
		}
	}
	joined := strings.Join(all, " ")
	if !strings.Contains(joined, "nodiscover") || !strings.Contains(joined, "4242") {
		t.Errorf("the command's opts must land in the all scope: %q", joined)
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

// TestPlan_HandoffTakesItsSizeFromTheProfile is the fix for X7.
//
// The handoff case carried "topology": {"bp": 4} and the composer refused it,
// so the case could not run at all. The number was wrong as well as unused: the
// profile sizes that network at one producer plus four validators, five nodes,
// and 4 was the validator count copied into a field that means something else.
//
// Removing it from the case leaves the size visible in exactly one place, and
// the plan is where a reader finds it.
func TestPlan_HandoffTakesItsSizeFromTheProfile(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
	  "binaries":{"from":"gwemix","to":"gwbft"},
	  "upgrade":{"profile":"../../profiles/wemix-upgrade.yaml","template":"t.json"}}`
	p := planFor(t, env, RunSuiteIn{})

	if p.Handoff.Producers != 1 || p.Handoff.Validators != 4 {
		t.Fatalf("roles = producers %d validators %d, want 1/4 from the profile",
			p.Handoff.Producers, p.Handoff.Validators)
	}
	if p.Handoff.AtFork != "croissant" || p.Handoff.ForkBlock != 20 {
		t.Errorf("fork = %s at %d", p.Handoff.AtFork, p.Handoff.ForkBlock)
	}
	if got := p.String(); !strings.Contains(got, "producers 1 · validators 4") {
		t.Errorf("the size must be on the plan:\n%s", got)
	}
}

// TestPlan_HandoffSurvivesAnUnreadableProfile: the plan is a display, and a
// display that fails for a reason the run is about to report more clearly only
// hides the real message.
func TestPlan_HandoffSurvivesAnUnreadableProfile(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
	  "binaries":{"from":"gwemix","to":"gwbft"},
	  "upgrade":{"profile":"no/such/profile.yaml","template":"t.json"}}`
	p := planFor(t, env, RunSuiteIn{})

	if p.Handoff == nil || p.Handoff.Profile != "no/such/profile.yaml" {
		t.Fatalf("handoff = %+v", p.Handoff)
	}
	if p.Handoff.Producers != 0 || p.Handoff.Validators != 0 {
		t.Error("an unread profile must leave the size unstated, not guessed")
	}
	if got := p.String(); strings.Contains(got, "producers") {
		t.Errorf("an unknown size must be omitted, not printed as zero:\n%s", got)
	}
}

// TestPlan_NamesWhoChoseEachValue walks one value through all three sources.
//
// The merge is the thing that loses provenance, so each case differs only in
// which layer names the value and asserts that the plan still knows. A value
// nobody named is the case worth holding hardest: a harness default is the one
// a reader cannot find by grepping their own files.
func TestPlan_NamesWhoChoseEachValue(t *testing.T) {
	const bare = `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet"}`
	const declared = `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":2},
	  "target":"/srv/net1","keys":{"nodekeys":{"ref":"keys/preset","source":"keyPreset"}}}`

	for _, tc := range []struct {
		name  string
		env   string
		in    RunSuiteIn
		field PlanField
		want  PlanSource
	}{
		{"binary from the command", declared, RunSuiteIn{Binary: "/bin/gstable"}, FieldBinary, SourceCommand},
		{"binary from the declaration", declared, RunSuiteIn{}, FieldBinary, SourceDeclaration},
		{"binary from the chain manifest", bare, RunSuiteIn{}, FieldBinary, SourceHarness},

		{"bp from the command", declared, RunSuiteIn{BPCount: 7}, FieldNodesBP, SourceCommand},
		{"bp from the declaration", declared, RunSuiteIn{}, FieldNodesBP, SourceDeclaration},
		{"bp from the harness default", bare, RunSuiteIn{}, FieldNodesBP, SourceHarness},

		{"keys dir from the command", declared, RunSuiteIn{KeysDir: "/k"}, FieldKeysDir, SourceCommand},
		{"keys dir from the declaration", declared, RunSuiteIn{}, FieldKeysDir, SourceDeclaration},
		{"keys dir from the harness default", bare, RunSuiteIn{}, FieldKeysDir, SourceHarness},

		{"keys source from the command", declared, RunSuiteIn{KeysSource: "generate"}, FieldKeysSource, SourceCommand},
		{"keys source from the declaration", declared, RunSuiteIn{}, FieldKeysSource, SourceDeclaration},
		{"keys source from the harness default", bare, RunSuiteIn{}, FieldKeysSource, SourceHarness},

		{"target from the declaration", declared, RunSuiteIn{}, FieldTarget, SourceDeclaration},
		{"target from the harness default", bare, RunSuiteIn{}, FieldTarget, SourceHarness},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := planFor(t, tc.env, tc.in)
			if got := p.From[tc.field]; got != tc.want {
				t.Fatalf("%s was chosen by %q, want %q", tc.field, got, tc.want)
			}
		})
	}
}

// TestPlan_SaysOnlyWhatTheDeclarationDidNotChoose keeps the rendered row short
// enough to read. Listing every field would put five words a reader already
// assumes in front of the one they need.
func TestPlan_SaysOnlyWhatTheDeclarationDidNotChoose(t *testing.T) {
	const env = `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":2},
	  "keys":{"nodekeys":{"ref":"keys/preset","source":"keyPreset"}}}`
	line := planFor(t, env, RunSuiteIn{}).chosenByLine()

	if strings.Contains(line, string(FieldBinary)) || strings.Contains(line, string(FieldKeysDir)) {
		t.Errorf("the declaration chose the binary and the keys, so the row must not mention them: %q", line)
	}
	// Nothing named a target, and that is exactly what the row is for.
	if !strings.Contains(line, string(FieldTarget)+": "+string(SourceHarness)) {
		t.Errorf("the harness chose the target and the row must say so: %q", line)
	}
}

// TestPlan_ATargetTheDeclarationNamedIsShown is the defect this found: the
// target row read the command's server selection alone, so a case whose
// env.target sent every node to another host printed "this machine". A plan
// that describes a different network than the one that launches is worse than
// no plan.
func TestPlan_ATargetTheDeclarationNamedIsShown(t *testing.T) {
	const env = `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":2},
	  "target":"ops@host.example:/data/net1"}`
	p := planFor(t, env, RunSuiteIn{})

	if strings.Contains(p.Target, "this machine") {
		t.Fatalf("the nodes run on another host, the plan says %q", p.Target)
	}
	if !strings.Contains(p.Target, "host.example") || !strings.Contains(p.Target, "/data/net1") {
		t.Errorf("the target must name the host and the root it was given: %q", p.Target)
	}
}

// TestRefuseMachineConflict covers the four shapes of "who picks the machine".
//
// The data root already refuses two answers rather than picking one. The
// machine had no such rule: a case declaring srv://alpha and a command passing
// --server beta composed on beta and said nothing — the plan printed "server
// beta" and credited the command, and alpha was gone. A test that runs on the
// wrong machine does not fail; it answers a question nobody asked.
func TestRefuseMachineConflict(t *testing.T) {
	alpha := resource.Spec{Server: "alpha", DataRoot: "/data/net1"}
	for _, tc := range []struct {
		name      string
		in        RunSuiteIn
		declared  resource.Spec
		wantError bool
	}{
		{"they disagree", RunSuiteIn{Server: resource.ServerRef{Name: "beta"}}, alpha, true},
		{"they agree", RunSuiteIn{Server: resource.ServerRef{Name: "alpha"}}, alpha, false},
		{"the command names none", RunSuiteIn{}, alpha, false},
		// A declaration naming a path has nothing to disagree with.
		{"the declaration names a path", RunSuiteIn{Server: resource.ServerRef{Name: "beta"}},
			resource.Spec{DataRoot: "/srv/net1"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := refuseMachineConflict(tc.in, tc.declared, "srv://alpha/data/net1")
			if tc.wantError && err == nil {
				t.Fatal("the run must be refused")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			// The refusal has to name both answers, because the reader has to
			// decide which one to delete.
			if tc.wantError && (!strings.Contains(err.Error(), "alpha") || !strings.Contains(err.Error(), "beta")) {
				t.Errorf("the refusal must name both machines: %v", err)
			}
		})
	}
}
