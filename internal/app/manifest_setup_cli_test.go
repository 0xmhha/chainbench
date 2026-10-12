package app

import (
	"slices"
	"testing"
)

// TestManifestSetupArgsComposeThroughChainUp: a manifest setup is one
// `chain up` that stops after init, on four producers of the selected chain,
// from the verified asset and the reviewed key preset.
func TestManifestSetupArgsComposeThroughChainUp(t *testing.T) {
	builtin := manifestSetupArgs("/setup/x", "/keys", "/assets/gstable", "stablenet", "", "")
	want := []string{"chain", "up", "--workspace-dir", "/setup/x", "--stage", "init", "--bp", "4", "--keys", "/keys", "--binary", "/assets/gstable", "--chain", "stablenet"}
	if !slices.Equal(builtin, want) {
		t.Fatalf("builtin args = %v, want %v", builtin, want)
	}
	external := manifestSetupArgs("/setup/x", "/keys", "/assets/gstable", "custom", "/setup/x/manifest.json", "/setup/x/genesis.json")
	if !slices.Contains(external, "--manifest") || !slices.Contains(external, "/setup/x/genesis.json") || slices.Contains(external, "--chain") {
		t.Fatalf("external args = %v, want the pinned manifest and template instead of --chain", external)
	}
}
