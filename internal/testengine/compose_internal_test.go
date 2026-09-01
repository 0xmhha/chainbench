package testengine

import (
	"context"
	"os"
	"path/filepath"
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
	if strings.Join(up.LaunchSet, ",") != "nodiscover,verbosity=4" && strings.Join(up.LaunchSet, ",") != "verbosity=4,nodiscover" {
		t.Errorf("launch set = %v", up.LaunchSet)
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
