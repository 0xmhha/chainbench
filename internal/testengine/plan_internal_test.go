package testengine

import (
	"github.com/0xmhha/chainbench/internal/core/origin"

	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/preflight"
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
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
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
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
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
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":2},
	  "launch":{"bp":{"mine":true}},
	  "config":{"node1":{"txpool.pricelimit":1}}}`
	p := planFor(t, env, RunSuiteIn{LaunchOpts: []string{"nodiscover"}, NetworkID: 4242})

	if got := p.Launch["bp"]; len(got) != 1 || !strings.HasPrefix(got[0].Knob, "mine") {
		t.Errorf("declared bp launch = %v", got)
	} else if got[0].From != origin.FromDeclaration {
		t.Errorf("the bp knob came from the declaration, not %q", got[0].From)
	}
	var all []string
	for _, k := range p.Launch["all"] {
		all = append(all, k.Knob)
		if k.From != origin.FromCommand {
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
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":2}}`
	p := planFor(t, env, RunSuiteIn{})
	if p.Keys.Source != keySourceKeyPreset {
		t.Fatalf("keys source = %q, want %q", p.Keys.Source, keySourceKeyPreset)
	}
	if p.Keys.Dir != defaultKeysDir {
		t.Errorf("keys dir = %q, want %q", p.Keys.Dir, defaultKeysDir)
	}
}

// TestPlan_AHardforkPlansItsLayoutLikeAnyOtherNetwork.
//
// An upgrade env used to plan as a handoff: no layout, a profile, and the size
// read out of that profile. Now its nodes are declared, so the plan shows the
// network the way it shows every other one — which is also the fix for X7, the
// case whose size lived in a document the composer refused to read.
func TestPlan_AHardforkPlansItsLayoutLikeAnyOtherNetwork(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"wemix",
	  "binaries":{"default":"gwemix","next":{"binary":"gwbft","chain":"wbft"}},
	  "upgrade":{"fork":"croissant","at":20,"from":"default","to":"next"},
	  "topology":{"nodes":[
	    {"index":1,"role":"en","binary":"next"},
	    {"index":2,"role":"en","binary":"next"},
	    {"index":3,"role":"bp"}
	  ]}}`
	t.Chdir("../..")
	p := planFor(t, env, RunSuiteIn{})

	if p.Nodes.BP != 1 || p.Nodes.EN != 2 {
		t.Fatalf("nodes = %+v, want the declared 1 bp + 2 en", p.Nodes)
	}
	rendered := p.String()
	if !strings.Contains(rendered, "next=gwbft") {
		t.Errorf("the plan must name the build that seals after the fork:\n%s", rendered)
	}
	if strings.Contains(rendered, "bp 0") {
		t.Errorf("the plan printed a layout it does not have:\n%s", rendered)
	}
}

// TestPlan_TargetNamesWhereTheNodesRun: the same specs compose a different
// network on a server set, and that is exactly the difference an operator wants
// confirmed before a remote run.
func TestPlan_TargetNamesWhereTheNodesRun(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
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

// TestPlan_NamesWhoChoseEachValue walks one value through all three sources.
//
// The merge is the thing that loses provenance, so each case differs only in
// which layer names the value and asserts that the plan still knows. A value
// nobody named is the case worth holding hardest: a harness default is the one
// a reader cannot find by grepping their own files.
func TestPlan_NamesWhoChoseEachValue(t *testing.T) {
	const bare = `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet"}`
	const declared = `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":2},
	  "target":"/srv/net1","keys":{"nodekeys":{"ref":"presets/keys","source":"keyPreset"}}}`

	for _, tc := range []struct {
		name  string
		env   string
		in    RunSuiteIn
		field PlanField
		want  origin.Origin
	}{
		{"binary from the command", declared, RunSuiteIn{Binary: "/bin/gstable"}, FieldBinary, origin.FromCommand},
		{"binary from the declaration", declared, RunSuiteIn{}, FieldBinary, origin.FromDeclaration},
		{"binary from the chain manifest", bare, RunSuiteIn{}, FieldBinary, origin.FromDefault},

		{"bp from the command", declared, RunSuiteIn{BPCount: 7}, FieldNodesBP, origin.FromCommand},
		{"bp from the declaration", declared, RunSuiteIn{}, FieldNodesBP, origin.FromDeclaration},
		{"bp from the harness default", bare, RunSuiteIn{}, FieldNodesBP, origin.FromDefault},

		{"keys dir from the command", declared, RunSuiteIn{KeysDir: "/k"}, FieldKeysDir, origin.FromCommand},
		{"keys dir from the declaration", declared, RunSuiteIn{}, FieldKeysDir, origin.FromDeclaration},
		{"keys dir from the harness default", bare, RunSuiteIn{}, FieldKeysDir, origin.FromDefault},

		{"keys source from the command", declared, RunSuiteIn{KeysSource: "generate"}, FieldKeysSource, origin.FromCommand},
		{"keys source from the declaration", declared, RunSuiteIn{}, FieldKeysSource, origin.FromDeclaration},
		{"keys source from the harness default", bare, RunSuiteIn{}, FieldKeysSource, origin.FromDefault},

		{"target from the declaration", declared, RunSuiteIn{}, FieldTarget, origin.FromDeclaration},
		{"target from the harness default", bare, RunSuiteIn{}, FieldTarget, origin.FromDefault},
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
	const env = `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":2},
	  "keys":{"nodekeys":{"ref":"presets/keys","source":"keyPreset"}}}`
	line := planFor(t, env, RunSuiteIn{}).chosenByLine()

	if strings.Contains(line, string(FieldBinary)) || strings.Contains(line, string(FieldKeysDir)) {
		t.Errorf("the declaration chose the binary and the keys, so the row must not mention them: %q", line)
	}
	// Nothing named a target, and that is exactly what the row is for.
	if !strings.Contains(line, string(FieldTarget)+": "+string(origin.FromDefault)) {
		t.Errorf("the harness chose the target and the row must say so: %q", line)
	}
}

// TestPlan_ATargetTheDeclarationNamedIsShown is the defect this found: the
// target row read the command's server selection alone, so a case whose
// env.target sent every node to another host printed "this machine". A plan
// that describes a different network than the one that launches is worse than
// no plan.
func TestPlan_ATargetTheDeclarationNamedIsShown(t *testing.T) {
	const env = `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
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

// TestPlan_NamesTheBinaryTheLaunchWillRun (W4b).
//
// A workspace-config exists to say where files actually are, and it puts
// binaries under the target's data root. The launch resolved a name through it
// and recorded /data/bin/gstable; the plan recorded "gstable". The pre-test
// comparison then refused a run that was entirely correct —
//
//	the network: asked for binary gstable, launched with /data/bin/gstable
//
// — and the refusal was right about the disagreement and wrong about who was
// at fault. The plan was reading one layer short.
func TestPlan_NamesTheBinaryTheLaunchWillRun(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "workspace-config.yaml")
	writeConfig(t, cfg, filepath.Join(dir, "out"))

	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":1}}`

	t.Run("placed through the environment file", func(t *testing.T) {
		p := planFor(t, env, RunSuiteIn{WorkspaceConfigPath: cfg})
		if want := "/data/bin/gstable"; p.Binary != want {
			t.Fatalf("plan binary = %q, want the placed %q", p.Binary, want)
		}
	})

	// Without one a name is the target's to resolve, and the plan must not
	// invent a path the launch would not use.
	t.Run("a name stays a name without one", func(t *testing.T) {
		p := planFor(t, env, RunSuiteIn{})
		if p.Binary != "gstable" {
			t.Fatalf("plan binary = %q, want gstable", p.Binary)
		}
	})

	// An operator's explicit path is already placed, on both sides.
	t.Run("an absolute path is left alone", func(t *testing.T) {
		p := planFor(t, env, RunSuiteIn{WorkspaceConfigPath: cfg, Binary: "/opt/gstable"})
		if p.Binary != "/opt/gstable" {
			t.Fatalf("plan binary = %q, want /opt/gstable", p.Binary)
		}
	})
}

// TestComposition_OneFormOfABinaryReference (W4c).
//
// Placing used to happen only inside init and start, so everything that asked
// a question ABOUT the launch asked it in the other language. Two of the three
// symptoms are here, on the path a run takes before anything launches.
func TestComposition_OneFormOfABinaryReference(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "workspace-config.yaml")
	writeConfig(t, cfg, filepath.Join(dir, "out"))

	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable","upgrade":"gstable-next"},
	  "topology":{"nodes":[
	    {"index":1,"role":"bp"},{"index":2,"role":"bp"},
	    {"index":3,"role":"bp","binary":"upgrade"},{"index":4,"role":"en"}]}}`

	spec := caseWithEnv(t, env)
	comp, err := compositionOf(context.Background(), spec,
		RunSuiteIn{WorkspaceConfigPath: cfg, DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compose: %v", err)
	}

	// The per-node map is handed to exec as it is stored. Stored as declared,
	// node 3 ran a bare name off the target's PATH while every other node in
	// the same network ran the placed path — two builds in one chain, silently,
	// whenever the PATH held a different one.
	t.Run("the per-node map holds placed paths", func(t *testing.T) {
		if got, want := comp.up.Binaries["upgrade"], "/data/bin/gstable-next"; got != want {
			t.Fatalf("binaries[upgrade] = %q, want %q", got, want)
		}
	})

	// Preflight compared the recorded placed path against the requested name,
	// so an identical request answered rebuild-all and the chain was recomposed
	// on every run — but only when the environment file was doing its job, which
	// is when an absolute --binary was not hiding it.
	t.Run("an identical request reuses what is composed", func(t *testing.T) {
		// Everything else answers what was asked, so the binary is the only
		// thing left that can disagree.
		have := preflight.Have{
			Chain: "stablenet", Binary: comp.up.Binary, KeysDir: comp.up.KeysDir,
			Started: true, Nodes: []preflight.Node{{Index: 1, PID: 100}},
		}
		d := preflight.Compare(have, chainsetup.WantOf(*comp.up))
		if d.Verdict != preflight.Reuse {
			t.Fatalf("verdict = %s, reasons %v — want reuse", d.Verdict, d.Reasons)
		}
	})
}

// TestComposition_ASecondGenesisForASecondBinary.
//
// A network can run two builds that do not accept the same genesis. The
// declaration names what the second one needs on top of the network's, and the
// composition carries an overlay file for it to the genesis step, which merges
// it onto the built genesis and records where that document landed.
func TestComposition_ASecondGenesisForASecondBinary(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable","next":"gstable-next"},
	  "genesis":{"mode":"template","perBinary":{
	     "next":{"set":{"config.croissantBlock":20}}}},
	  "topology":{"nodes":[
	    {"index":1,"role":"bp"},{"index":2,"role":"bp","binary":"next"}]}}`

	spec := caseWithEnv(t, env)
	if got := spec.Chain.GenesisPerBinary["next"]; got == nil {
		t.Fatalf("the declaration did not reach the spec: %+v", spec.Chain)
	}
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	path := comp.up.GenesisPerBinary["next"]
	if path == "" {
		t.Fatalf("no overlay was rendered for the second binary: %v", comp.up.GenesisPerBinary)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "croissantBlock") {
		t.Errorf("the rendered overlay does not carry what was declared: %s", b)
	}
}

// TestComposition_ASecondGenesisMustNameADeclaredBinary: the nodes meant to get
// it would otherwise initialize from the network's genesis and nothing would
// say so, which is the shape a misspelled name takes.
func TestComposition_ASecondGenesisMustNameADeclaredBinary(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable","next":"gstable-next"},
	  "genesis":{"mode":"template","perBinary":{"nxet":{"set":{"config.croissantBlock":20}}}},
	  "topology":{"bp":2}}`
	raw := `{"schemaVersion":"2","kind":"case","id":"typo","chainPreset":` + env + `,
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	if _, err := dsl.Parse([]byte(raw)); err == nil ||
		!strings.Contains(err.Error(), "binaries does not declare") {
		t.Fatalf("a misspelled binary name was accepted: %v", err)
	}
}

// TestComposition_TheDeclaredDefaultIsWhatTheRestOfTheNetworkRuns.
//
// A node table that assigns binaries to SOME nodes leaves the others on the
// declaration's "default". They used to be put on whichever binary the first
// assigned node happened to name, so a four-node network with two on the
// successor ran all four on the successor — and the plan said so, with nothing
// about it looking wrong.
func TestComposition_TheDeclaredDefaultIsWhatTheRestOfTheNetworkRuns(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable","next":"gstable-next"},
	  "topology":{"nodes":[
	    {"index":1,"role":"bp"},{"index":2,"role":"bp"},
	    {"index":3,"role":"bp","binary":"next"},{"index":4,"role":"bp","binary":"next"}]}}`

	spec := caseWithEnv(t, env)
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	if got := comp.up.Binary; got != "gstable" {
		t.Errorf("the network's binary = %q, want the declared default %q", got, "gstable")
	}
	if got := comp.up.Binaries["next"]; got != "gstable-next" {
		t.Errorf("binaries[next] = %q, want gstable-next", got)
	}
}

// TestComposition_ABinaryCanNameItsOwnChain.
//
// A network that runs two builds of two different chains needs each node
// configured against its own: the chain is what says the flag vocabulary, the
// RPC namespace and what the consensus asks of a launch. The declaration says
// it per binary, and a bare string keeps meaning "this environment's chain".
func TestComposition_ABinaryCanNameItsOwnChain(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"wemix",
	  "binaries":{
	    "default":"gwemix",
	    "next":{"binary":"gwbft","chain":"wbft"}},
	  "topology":{"nodes":[
	    {"index":1,"role":"bp"},{"index":2,"role":"en","binary":"next"}]}}`

	spec := caseWithEnv(t, env)
	if got := spec.Chain.BinaryChains["next"]; got != "wbft" {
		t.Fatalf("the declaration did not reach the spec: %v", spec.Chain.BinaryChains)
	}
	if _, named := spec.Chain.BinaryChains["default"]; named {
		t.Errorf("a bare string named a chain: %v", spec.Chain.BinaryChains)
	}
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	if got := comp.up.BinaryChains["next"]; got != "wbft" {
		t.Errorf("the request carries %v, want next->wbft", comp.up.BinaryChains)
	}
}

// TestComposition_ABinaryEntryNeedsABinary: an object form that names only a
// chain says which chain nothing runs.
func TestComposition_ABinaryEntryNeedsABinary(t *testing.T) {
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"wemix",
	  "binaries":{"default":"gwemix","next":{"chain":"wbft"}},
	  "topology":{"bp":1}}`
	raw := `{"schemaVersion":"2","kind":"case","id":"c","chainPreset":` + env + `,
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	if _, err := dsl.Parse([]byte(raw)); err == nil ||
		!strings.Contains(err.Error(), "needs a binary") {
		t.Fatalf("an entry with no binary was accepted: %v", err)
	}
}

// TestComposition_TheDeclaredDefaultIsExpandedLikeAnyOtherReference.
//
// A declaration writes ${GSTABLE_BIN:-gstable} for the network's binary as
// readily as for a per-node one. The per-node entries were expanded and this
// lookup was not, so the plan printed the placeholder and exec would have been
// handed it — found by composing a network that used the object form for both.
func TestComposition_TheDeclaredDefaultIsExpandedLikeAnyOtherReference(t *testing.T) {
	t.Setenv("CB_TEST_DEFAULT_BIN", "/opt/gstable")
	env := `{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet",
	  "binaries":{"default":"${CB_TEST_DEFAULT_BIN}","next":"gstable-next"},
	  "topology":{"nodes":[
	    {"index":1,"role":"bp"},{"index":2,"role":"bp","binary":"next"}]}}`

	spec := caseWithEnv(t, env)
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	if got := comp.up.Binary; got != "/opt/gstable" {
		t.Errorf("the network's binary = %q, want the expanded /opt/gstable", got)
	}
}
