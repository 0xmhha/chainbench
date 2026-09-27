package testengine

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// TestResolveComposition_RefusesToComposeForAnAttachCase: a case that declares
// env.attach runs against a network that is already up and composes none. A
// --workspace-dir run used to compose a default network for it anyway, so
// basic/08 — the case that checks attaching by declaration — passed without
// ever attaching.
func TestResolveComposition_RefusesToComposeForAnAttachCase(t *testing.T) {
	spec, err := filepath.Abs("../../tests/tc/basic/08-attached-chain-produces.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = resolveComposition(context.Background(), RunSuiteIn{
		SpecPaths: []string{spec}, DataDir: t.TempDir(), Binary: "/opt/gstable",
	})
	if err == nil {
		t.Fatal("an attach-only case was composed for")
	}
	for _, want := range []string{"declares env.attach", "without --workspace-dir", "--chain-preset"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not say %q: %v", want, err)
		}
	}
}

// TestResolveComposition_APresetOverrideComposesAnAttachCase: --chain-preset
// replaces the declaration, attach and all, so the run composes the network the
// operator named — the documented way to run such a case on a fresh network.
func TestResolveComposition_APresetOverrideComposesAnAttachCase(t *testing.T) {
	spec, err := filepath.Abs("../../tests/tc/basic/08-attached-chain-produces.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _, comp, err := resolveComposition(context.Background(), RunSuiteIn{
		SpecPaths: []string{spec}, DataDir: t.TempDir(), Binary: "/opt/gstable", Env: "stablenet-bp4",
	})
	if err != nil {
		t.Fatalf("a preset override was refused: %v", err)
	}
	if comp.up == nil || comp.up.BPCount != 4 {
		t.Errorf("composition = %+v, want the bp4 network the preset names", comp.up)
	}
}
