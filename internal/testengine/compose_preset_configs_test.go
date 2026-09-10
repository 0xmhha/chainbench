package testengine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// TestApplyPresetConfigs_MapsLogicalNames: a node table's config value that is a
// preset config name resolves to that preset's file; a direct reference and an
// empty value are left alone.
func TestApplyPresetConfigs_MapsLogicalNames(t *testing.T) {
	up := &chainsetup.NetUpIn{Topology: &node.Topology{
		Chain: "stablenet",
		Nodes: []node.Entry{
			{Index: 1, Role: "bp", Config: "validator"},        // logical name -> mapped
			{Index: 2, Role: "en", Config: "endpoint"},         // logical name -> mapped
			{Index: 3, Role: "en", Config: "/abs/direct.toml"}, // direct ref -> unchanged
			{Index: 4, Role: "en"},                             // none -> unchanged
		},
	}}
	preset := resource.InputPreset{Configs: map[string]string{
		"validator": "srv://server-01/data/configs/v.toml",
		"endpoint":  "en.toml", // portable reference under the configs purpose
	}}

	applyPresetConfigs(up, preset)

	want := []string{
		"srv://server-01/data/configs/v.toml",
		"en.toml",
		"/abs/direct.toml",
		"",
	}
	for i, w := range want {
		if got := up.Topology.Nodes[i].Config; got != w {
			t.Fatalf("node%d config = %q, want %q", i+1, got, w)
		}
	}
}

// TestCompositionOf_PresetConfigsResolveThroughTheDSL proves the mapping is
// wired: a topology names its nodes' configs logically, and inputs.mode=prepared
// with a preset resolves those names to files on the composed node table.
func TestCompositionOf_PresetConfigsResolveThroughTheDSL(t *testing.T) {
	dir := t.TempDir()
	wcPath := filepath.Join(dir, "workspace-config.yaml")
	body := "version: 1\ndataRoot: /data\n" +
		"paths: {binaries: bin, configs: configs, genesis: genesis, keystore: keystore, keyrings: keys, nodes: node, runtime: runtime, logs: logs}\n" +
		"control: {artifactRoot: ~/.chainbench}\n" +
		"inputs: {mode: prepared, preset: regression}\nexecution: {chain: fresh}\n" +
		"presets:\n  regression:\n    configs:\n      validator: srv://server-01/data/configs/v.toml\n"
	if err := os.WriteFile(wcPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	env := `{"schemaVersion":"2","kind":"env","id":"e","chain":"stablenet","binaries":{"default":"gstable"},` +
		`"topology":{"chain":"stablenet","nodes":[` +
		`{"index":1,"role":"bp","config":"validator"},` +
		`{"index":2,"role":"bp","config":"validator"},` +
		`{"index":3,"role":"bp","config":"validator"},` +
		`{"index":4,"role":"bp","config":"validator"}]}}`
	spec := caseWithEnv(t, env)

	comp, err := compositionOf(context.Background(), spec, RunSuiteIn{DataDir: dir, WorkspaceConfigPath: wcPath})
	if err != nil {
		t.Fatalf("compositionOf: %v", err)
	}
	if comp.up.Topology == nil {
		t.Fatal("expected an inline topology")
	}
	for i, n := range comp.up.Topology.Nodes {
		if n.Config != "srv://server-01/data/configs/v.toml" {
			t.Fatalf("node%d config = %q, want the preset file", i+1, n.Config)
		}
	}
}

// TestApplyPresetConfigs_NoTopologyOrNoMapIsANoop: with no node table, or no
// config map, there is nothing to resolve and nothing changes.
func TestApplyPresetConfigs_NoTopologyOrNoMapIsANoop(t *testing.T) {
	// No topology.
	up := &chainsetup.NetUpIn{}
	applyPresetConfigs(up, resource.InputPreset{Configs: map[string]string{"a": "b"}})
	if up.Topology != nil {
		t.Fatal("no topology must stay nil")
	}
	// No config map.
	up = &chainsetup.NetUpIn{Topology: &node.Topology{Nodes: []node.Entry{{Index: 1, Config: "validator"}}}}
	applyPresetConfigs(up, resource.InputPreset{})
	if up.Topology.Nodes[0].Config != "validator" {
		t.Fatalf("no config map must leave the value: %q", up.Topology.Nodes[0].Config)
	}
}
