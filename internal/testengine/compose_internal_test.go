package testengine

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// caseWithEnv builds a v2 case whose env is the given object, parsed the way
// the suite parses it.
func caseWithEnv(t *testing.T, env string) dsl.Spec {
	t.Helper()
	raw := `{"schemaVersion":"2","kind":"case","id":"c","env":` + env + `,
	  "steps":[{"expect":"blockNumber","compare":"Greater","is":"0"}]}`
	s, err := dsl.Parse([]byte(raw))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return s
}

func TestCompositionOf_WorkspaceFromDeclaration(t *testing.T) {
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},
	  "topology":{"bp":3,"en":1,"syncMode":"snap"},
	  "keys":{"nodekeys":{"source":"generate","ref":"/keys/gen"}},
	  "hardforks":{"boho":10},
	  "launch":{"all":{"nodiscover":true,"verbosity":"4"}},
	  "genesis":{"set":{"config.chainId":77}}}`)
	dir := t.TempDir()
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: dir})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.handoff != nil || comp.up == nil {
		t.Fatal("a single-binary env composes through the workspace")
	}
	up := comp.up
	if up.Chain != "stablenet" || up.Binary != "gstable" || up.Stage != chainsetup.UpStart {
		t.Errorf("chain/binary/stage = %q/%q/%q", up.Chain, up.Binary, up.Stage)
	}
	if up.Validators != 3 || up.Endpoints != 1 || up.EndpointSyncMode != "snap" {
		t.Errorf("topology = %d/%d/%q, want 3/1/snap", up.Validators, up.Endpoints, up.EndpointSyncMode)
	}
	if up.KeysDir != "/keys/gen" || up.KeysSource != "generate" {
		t.Errorf("keys = %q/%q", up.KeysDir, up.KeysSource)
	}
	if strings.Join(up.GenesisSet, ",") != "bohoBlock=10" {
		t.Errorf("genesis set = %v", up.GenesisSet)
	}
	// The env's launch knobs travel scoped ("all" here), not in the flat set.
	all := append([]string(nil), up.LaunchScoped["all"]...)
	sort.Strings(all)
	if strings.Join(all, ",") != "nodiscover=true,verbosity=4" {
		t.Errorf("launch scoped[all] = %v", up.LaunchScoped["all"])
	}
	if up.OverlayPath == "" {
		t.Fatal("a declared genesis set must reach the genesis step as an overlay file")
	}
	b, err := os.ReadFile(up.OverlayPath)
	if err != nil || !strings.Contains(string(b), `"chainId": 77`) {
		t.Errorf("overlay file = %s (%v)", b, err)
	}
	if filepath.Dir(up.OverlayPath) != dir {
		t.Errorf("overlay written outside the workspace: %s", up.OverlayPath)
	}
}

func TestCompositionOf_OverridesAndDefaults(t *testing.T) {
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet","binaries":{"default":"gstable"}}`)
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir(), Binary: "/opt/gstable", Validators: 5, KeysDir: "/k"})
	if err != nil {
		t.Fatal(err)
	}
	if comp.up.Binary != "/opt/gstable" || comp.up.Validators != 5 || comp.up.KeysDir != "/k" {
		t.Errorf("overrides not applied: %+v", comp.up)
	}
	comp, err = compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if comp.up.Validators != suiteDefaultValidators || comp.up.KeysDir != defaultKeysDir || comp.up.OverlayPath != "" {
		t.Errorf("defaults: %+v", comp.up)
	}
	if _, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir(), Chain: "wbft"}); err == nil {
		t.Error("a request naming another chain than the spec must be refused")
	}
}

func TestCompositionOf_HandoffFromDeclaration(t *testing.T) {
	t.Setenv("HANDOFF_TEMPLATE", "/tmpl/genesis-template.json")
	t.Setenv("GWBFT_BIN", "")
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
	  "binaries":{"producer":"gwemix","validator":"${GWBFT_BIN:-gwbft}"},
	  "upgrade":{"profile":"profiles/wemix-upgrade.yaml","template":"${HANDOFF_TEMPLATE}"}}`)
	dir := t.TempDir()
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: dir})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.up != nil || comp.handoff == nil {
		t.Fatal("an upgrade env composes as a handoff")
	}
	h := comp.handoff
	if h.FromBinary != "gwemix" || h.ToBinary != "gwbft" {
		t.Errorf("binaries = %q -> %q (a ${VAR:-default} with the var unset takes the default)", h.FromBinary, h.ToBinary)
	}
	if h.Template != "/tmpl/genesis-template.json" || h.ProfilePath != "profiles/wemix-upgrade.yaml" || h.DataDir != dir {
		t.Errorf("handoff inputs = %+v", h)
	}
	if h.PresetDir != defaultKeysDir {
		t.Errorf("preset = %q", h.PresetDir)
	}
	if _, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: dir, Binary: "/x"}); err == nil {
		t.Error("--binary has no role in a handoff and must be refused")
	}
}

func TestCompositionOf_NodeTablePerNodeBinary(t *testing.T) {
	// Two binaries of the same family run side by side, declared per node.
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
	  "binaries":{"stable":"/opt/gstable","wbft":"/opt/gwbft"},
	  "topology":{"nodes":[
	    {"role":"bp","binary":"stable"},
	    {"role":"bp","binary":"wbft"},
	    {"role":"en","binary":"wbft","sync":"snap"}
	  ]}}`)
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.handoff != nil || comp.up == nil {
		t.Fatal("a same-family node table composes through the workspace")
	}
	up := comp.up
	if up.Topology == nil || len(up.Topology.Nodes) != 3 {
		t.Fatalf("node table not threaded: %+v", up.Topology)
	}
	if up.Topology.Nodes[0].Binary != "stable" || up.Topology.Nodes[2].SyncMode != "snap" {
		t.Errorf("per-node fields lost: %+v", up.Topology.Nodes)
	}
	if up.Binaries["stable"] != "/opt/gstable" || up.Binaries["wbft"] != "/opt/gwbft" {
		t.Errorf("binaries not resolved: %v", up.Binaries)
	}
	// No count is set; the node table is the sizing.
	if up.Validators != 0 || up.Endpoints != 0 {
		t.Errorf("counts leaked with a node table: %d/%d", up.Validators, up.Endpoints)
	}
	// The fallback binary is the first node's, for any node naming none.
	if up.Binary != "/opt/gstable" {
		t.Errorf("fallback binary = %q, want the first node's", up.Binary)
	}
}

// TestCompositionOf_GenerateDefaultsToWorkspaceDir pins S5's key-reuse edge:
// a spec that asks to generate keys but names no ref must not default to the
// shared preset (where GeneratedKeys would reuse the preset's identities and
// fail once the network wants more). It goes to a workspace-local dir instead;
// every other source still defaults to the shared preset.
func TestCompositionOf_GenerateDefaultsToWorkspaceDir(t *testing.T) {
	dir := t.TempDir()
	gen := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},
	  "keys":{"nodekeys":{"source":"generate"}}}`)
	comp, err := compositionOf(context.Background(), gen, RunSuiteIn{DataDir: dir})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if want := filepath.Join(dir, generatedKeysSubdir); comp.up.KeysDir != want {
		t.Errorf("generate keys dir = %q, want the workspace-local %q", comp.up.KeysDir, want)
	}
	if comp.up.KeysSource != keySourceGenerate {
		t.Errorf("keys source = %q, want generate", comp.up.KeysSource)
	}

	// Preset (the default source) still composes from the shared preset.
	preset := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet","binaries":{"default":"gstable"}}`)
	pc, err := compositionOf(context.Background(), preset, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if pc.up.KeysDir != defaultKeysDir {
		t.Errorf("preset keys dir = %q, want %q", pc.up.KeysDir, defaultKeysDir)
	}
}

// TestCompositionOf_NodeTablePerNodeConfig pins S4: a node may name a
// pre-written config file, and it rides the node table to the composition
// (where the config step later writes it verbatim). The path is expanded like
// every other declared path.
func TestCompositionOf_NodeTablePerNodeConfig(t *testing.T) {
	t.Setenv("CFGDIR", "/opt/cfg")
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},
	  "topology":{"nodes":[
	    {"role":"bp","config":"${CFGDIR}/node1.toml"},
	    {"role":"en"}
	  ]}}`)
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.up == nil || comp.up.Topology == nil || len(comp.up.Topology.Nodes) != 2 {
		t.Fatalf("node table not threaded: %+v", comp.up)
	}
	if got := comp.up.Topology.Nodes[0].Config; got != "/opt/cfg/node1.toml" {
		t.Errorf("node1 config = %q, want the expanded path", got)
	}
	if comp.up.Topology.Nodes[1].Config != "" {
		t.Errorf("node2 config = %q, want empty", comp.up.Topology.Nodes[1].Config)
	}
}

// TestCompositionOf_NodeTablePerNodeKey pins S5's per-node key surface: a node
// may name its key (a file path or 0x-hex) and it rides the node table to the
// composition, where the keys step makes it the node's identity. The path form
// is expanded like every other declared path.
func TestCompositionOf_NodeTablePerNodeKey(t *testing.T) {
	t.Setenv("KEYDIR", "/opt/keys")
	dir := t.TempDir()
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},
	  "topology":{"nodes":[
	    {"role":"bp","key":"0xabc"},
	    {"role":"bp","key":"${KEYDIR}/node2.key"},
	    {"role":"en"}
	  ]}}`)
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: dir})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	// A pinned key builds a fresh set, so the keys land workspace-local, never
	// in the shared preset (where the build would be blocked by reuse).
	if want := filepath.Join(dir, generatedKeysSubdir); comp.up.KeysDir != want {
		t.Errorf("keys dir = %q, want the workspace-local %q for a keyed node table", comp.up.KeysDir, want)
	}
	if comp.up == nil || comp.up.Topology == nil || len(comp.up.Topology.Nodes) != 3 {
		t.Fatalf("node table not threaded: %+v", comp.up)
	}
	if got := comp.up.Topology.Nodes[0].Key; got != "0xabc" {
		t.Errorf("node1 key = %q, want the inline hex", got)
	}
	if got := comp.up.Topology.Nodes[1].Key; got != "/opt/keys/node2.key" {
		t.Errorf("node2 key = %q, want the expanded path", got)
	}
	if comp.up.Topology.Nodes[2].Key != "" {
		t.Errorf("node3 key = %q, want empty", comp.up.Topology.Nodes[2].Key)
	}
}

// TestCompositionOf_WorkspaceConfigDataRootConflict pins the locality/root
// consistency check: the workspace-config is the one owner of the data root, so
// an env target that names a different root is a conflict, not a silent
// override. A matching root (or no env target) is accepted.
func TestCompositionOf_WorkspaceConfigDataRootConflict(t *testing.T) {
	dir := t.TempDir()
	wcPath := filepath.Join(dir, "workspace-config.yaml")
	writeWC := func(root string) {
		body := "version: 1\ndataRoot: " + root + "\n" +
			"paths: {binaries: bin, configs: configs, genesis: genesis, keystore: keystore, keyrings: keys, nodes: node, runtime: runtime, logs: logs}\n" +
			"control: {artifactRoot: ~/.chainbench}\ninputs: {mode: generated}\nexecution: {chain: fresh}\n"
		if err := os.WriteFile(wcPath, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// env.target names a different root than the workspace-config -> conflict.
	writeWC("/data")
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet","binaries":{"default":"gstable"},"target":"/other/root"}`)
	_, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: dir, WorkspaceConfigPath: wcPath})
	if err == nil || !strings.Contains(err.Error(), "data root conflict") {
		t.Fatalf("mismatched roots must conflict, got %v", err)
	}

	// The same root is fine.
	writeWC("/other/root")
	if _, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: dir, WorkspaceConfigPath: wcPath}); err != nil {
		t.Fatalf("matching roots must be accepted: %v", err)
	}

	// No env target: the workspace-config root is used with no conflict.
	writeWC("/data")
	plain := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet","binaries":{"default":"gstable"}}`)
	comp, err := compositionOf(context.Background(), plain, RunSuiteIn{DataDir: dir, WorkspaceConfigPath: wcPath})
	if err != nil {
		t.Fatalf("no env target must be accepted: %v", err)
	}
	if comp.up.Target.DataRoot != "/data" {
		t.Fatalf("data root = %q, want the workspace-config's /data", comp.up.Target.DataRoot)
	}
}

// TestCompositionOf_NodeTablePnSelectsProxied pins WA9: a pn declared in a node
// table means the same proxy tier as a pn in the count form, so the composition
// must select proxied peering. Under mesh the tier would do nothing and
// endpoints would dial producers directly.
func TestCompositionOf_NodeTablePnSelectsProxied(t *testing.T) {
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},
	  "topology":{"nodes":[
	    {"role":"bp"},
	    {"role":"pn"},
	    {"role":"en"}
	  ]}}`)
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.up == nil {
		t.Fatal("a node table composes through the workspace")
	}
	if comp.up.Peering != "proxied" {
		t.Errorf("peering = %q, want proxied for a node table that declares a pn", comp.up.Peering)
	}
}

// TestCompositionOf_SurfaceDefaultsConverge pins the E9 parity guarantee: both
// the CLI and MCP pass zero-values when a knob is unset, so compositionOf is the
// single source of the canonical defaults (validators, keys). Passing the
// explicit default must equal passing nothing — otherwise the two surfaces could
// drift on a default. The constants are pinned so a surface's flag/arg default
// cannot silently diverge from the seam.
func TestCompositionOf_SurfaceDefaultsConverge(t *testing.T) {
	if suiteDefaultValidators != 4 || defaultKeysDir != "keys/preset" {
		t.Fatalf("canonical defaults drifted: validators=%d keys=%q", suiteDefaultValidators, defaultKeysDir)
	}
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet","binaries":{"default":"gstable"}}`)

	// Unset (what both surfaces pass when the operator/agent gives nothing).
	unset, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	// The explicit default a surface might pass instead.
	explicit, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir(), KeysDir: defaultKeysDir})
	if err != nil {
		t.Fatal(err)
	}
	if unset.up.KeysDir != explicit.up.KeysDir || unset.up.KeysDir != "keys/preset" {
		t.Errorf("keys default diverges: unset=%q explicit=%q", unset.up.KeysDir, explicit.up.KeysDir)
	}
	if unset.up.Validators != suiteDefaultValidators {
		t.Errorf("validators default = %d, want %d", unset.up.Validators, suiteDefaultValidators)
	}
}

func TestInlineTopologyOf_Rejects(t *testing.T) {
	bins := map[string]string{"wbft": "/opt/gwbft"}
	cases := map[string]string{
		"undeclared binary": `{"nodes":[{"role":"bp","binary":"ghost"}]}`,
		"missing role":      `{"nodes":[{"binary":"wbft"}]}`,
		"unknown key":       `{"nodes":[{"role":"bp","pn":true}]}`,
		"empty list":        `{"nodes":[]}`,
		"no producer":       `{"nodes":[{"role":"en"}]}`,
	}
	for name, topoJSON := range cases {
		t.Run(name, func(t *testing.T) {
			spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
			  "binaries":{"wbft":"/opt/gwbft"},"topology":`+topoJSON+`}`)
			_ = bins
			if _, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()}); err == nil {
				t.Fatalf("topology %s accepted", topoJSON)
			}
		})
	}
}

func TestTopologyOf_RejectsWhatItDoesNotKnow(t *testing.T) {
	cases := map[string]map[string]any{
		"unknown key":  {"boot": 1},
		"fraction":     {"bp": 2.5},
		"negative":     {"en": -1},
		"not a number": {"bp": "four"},
		"sync not str": {"syncMode": 3},
	}
	for name, topo := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, _, _, _, err := topologyOf(topo); err == nil {
				t.Fatalf("topology %v accepted", topo)
			}
		})
	}
	v, e, p, m, auto, err := topologyOf(map[string]any{"validators": float64(4), "endpoints": float64(2), "pn": float64(1), "sync_mode": "archive"})
	if err != nil || v != 4 || e != 2 || p != 1 || m != "archive" || auto {
		t.Fatalf("got %d/%d/%d/%q auto=%v (%v)", v, e, p, m, auto, err)
	}
	// bp: "max" leaves the count for the composer to fill and flags autoBP;
	// pn/en still parse alongside it.
	v, e, p, _, auto, err = topologyOf(map[string]any{"bp": "max", "pn": float64(1), "en": float64(1)})
	if err != nil || !auto || v != 0 || p != 1 || e != 1 {
		t.Fatalf("bp:max got v=%d e=%d p=%d auto=%v (%v)", v, e, p, auto, err)
	}
}

func TestSameChain(t *testing.T) {
	a := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet","binaries":{"default":"g"}}`)
	b := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"f","chain":"wbft","binaries":{"default":"g"}}`)
	if err := sameChain([]dsl.Spec{a, a}); err != nil {
		t.Errorf("same chain refused: %v", err)
	}
	if err := sameChain([]dsl.Spec{a, b}); err == nil {
		t.Error("two chains in one suite accepted")
	}
}

func TestExpand_DefaultsAndVars(t *testing.T) {
	t.Setenv("CB_X", "set")
	t.Setenv("CB_EMPTY", "")
	for in, want := range map[string]string{
		"$CB_X":                 "set",
		"${CB_X}/bin":           "set/bin",
		"${CB_EMPTY:-fallback}": "fallback",
		"${CB_X:-fallback}":     "set",
		"plain":                 "plain",
	} {
		if got := expand(in); got != want {
			t.Errorf("expand(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestCompositionOf_EnvTargetPlaces pins WA19: the env's target selects where
// the network is placed, threaded into the composition rather than only feeding
// the reuse fingerprint (which left a declared target moving the key but not the
// nodes).
func TestCompositionOf_EnvTargetPlaces(t *testing.T) {
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"target":"srv://bp1/data"}`)
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.up == nil || comp.up.Target.Server != "bp1" || comp.up.Target.DataRoot != "/data" {
		t.Fatalf("env target not threaded to placement: %+v", comp.up.Target)
	}

	// A malformed target fails composition rather than being ignored.
	bad := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"target":"srv://"}`)
	if _, err := compositionOf(context.Background(), bad, RunSuiteIn{DataDir: t.TempDir()}); err == nil {
		t.Fatal("a malformed env target must fail composition")
	}
}

// TestCompositionOf_HandoffRejectsEnvComposeFields pins WA20: a handoff composes
// from its profile and template, so env-level topology/hardforks/launch/config
// have nowhere to go and are refused rather than silently dropped.
func TestCompositionOf_HandoffRejectsEnvComposeFields(t *testing.T) {
	t.Setenv("HANDOFF_TEMPLATE", "/tmpl/g.json")
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"wbft",
	  "binaries":{"producer":"gwemix","validator":"gwbft"},
	  "topology":{"validators":4},
	  "upgrade":{"profile":"p.yaml","template":"${HANDOFF_TEMPLATE}"}}`)
	if _, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()}); err == nil {
		t.Fatal("a handoff env that also declares a topology must be refused, not silently dropped")
	}
}

// TestCompositionOf_EnvManifestThreads pins WA21: an env's manifest and genesis
// template reach the composition, so a DSL spec can run an external chain on a
// built-in family — the capability was CLI-only before.
func TestCompositionOf_EnvManifestThreads(t *testing.T) {
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},
	  "manifest":"/chains/acme.json","genesisTemplate":"/chains/acme-genesis.json"}`)
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.up == nil || comp.up.ManifestPath != "/chains/acme.json" || comp.up.TemplatePath != "/chains/acme-genesis.json" {
		t.Fatalf("manifest/template not threaded: %+v", comp.up)
	}
}

// TestCompositionOf_EnvBlueprintAndKeysValidators pins WA21 (E): an env can name
// a blueprint (layout + keys in one document) and, for a generated key set, how
// many identities join the validator set — both threaded to the composition,
// where they were CLI-only before.
func TestCompositionOf_EnvBlueprintAndKeysValidators(t *testing.T) {
	// Blueprint: threaded to the composition's BlueprintPath.
	bp := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"blueprint":"/net/blueprint.yaml"}`)
	comp, err := compositionOf(context.Background(), bp, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.up == nil || comp.up.BlueprintPath != "/net/blueprint.yaml" {
		t.Fatalf("blueprint not threaded: %+v", comp.up)
	}

	// keys.validators: threaded to the composition's KeysValidators.
	kv := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet",
	  "binaries":{"default":"gstable"},"topology":{"bp":3},
	  "keys":{"nodekeys":{"source":"generate","ref":"keys/gen","validators":2}}}`)
	comp, err = compositionOf(context.Background(), kv, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.up == nil || comp.up.KeysValidators != 2 {
		t.Fatalf("keys validators not threaded: %+v", comp.up)
	}
}

// TestCompositionOf_WemixProxiedPn pins S1: poa (wemix) now accepts a pn, so a
// wemix bp/pn/en topology composes as the proxied graph. The pn is a
// non-producing discovery hub; validators stay the bp set. Verified live on the
// docker fleet (a wemix bp3/pn1/en1 network comes up and produces blocks).
func TestCompositionOf_WemixProxiedPn(t *testing.T) {
	spec := caseWithEnv(t, `{"schemaVersion":"2","kind":"env","id":"e","chain":"wemix",
	  "binaries":{"default":"gwemix"},"topology":{"bp":3,"pn":1,"en":1}}`)
	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: t.TempDir()})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.up == nil || comp.up.Peering != "proxied" {
		t.Fatalf("peering = %q, want proxied for a wemix pn topology", comp.up.Peering)
	}
}
