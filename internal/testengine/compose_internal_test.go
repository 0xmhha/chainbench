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
		"unknown key":  {"pn": 1},
		"fraction":     {"bp": 2.5},
		"negative":     {"en": -1},
		"not a number": {"bp": "four"},
		"sync not str": {"syncMode": 3},
	}
	for name, topo := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, _, err := topologyOf(topo); err == nil {
				t.Fatalf("topology %v accepted", topo)
			}
		})
	}
	v, e, m, err := topologyOf(map[string]any{"validators": float64(4), "endpoints": float64(2), "sync_mode": "archive"})
	if err != nil || v != 4 || e != 2 || m != "archive" {
		t.Fatalf("got %d/%d/%q (%v)", v, e, m, err)
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
