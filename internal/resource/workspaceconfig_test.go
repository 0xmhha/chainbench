package resource

import (
	"path/filepath"
	"strings"
	"testing"
)

const validConfig = `
version: 1
dataRoot: /data
paths:
  binaries: bin
  configs: configs
  genesis: genesis
  keystore: keystore
  keyrings: keys
  nodes: node
  runtime: runtime
  logs: logs
binaryAliases:
  gwemix: linux-amd64/gwemix
control:
  artifactRoot: ~/.chainbench
inputs:
  mode: generated
execution:
  chain: fresh
`

func TestParseWorkspaceConfig_Valid(t *testing.T) {
	c, err := ParseWorkspaceConfig([]byte(validConfig))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if c.Version != 1 || c.DataRoot != "/data" {
		t.Fatalf("version/dataRoot = %d/%q", c.Version, c.DataRoot)
	}
	if c.Paths.Binaries != "bin" || c.Paths.Nodes != "node" || c.Paths.Runtime != "runtime" {
		t.Fatalf("paths not parsed: %+v", c.Paths)
	}
	if c.Inputs.Mode != InputGenerated || c.Execution.Chain != ChainFresh {
		t.Fatalf("inputs/execution = %q/%q", c.Inputs.Mode, c.Execution.Chain)
	}
}

// TestResolve_JoinsPurposeAndRef is the core path contract: dataRoot / paths[use] / ref,
// POSIX-joined, so the same ref resolves under whatever root the environment names.
func TestResolve_JoinsPurposeAndRef(t *testing.T) {
	c, err := ParseWorkspaceConfig([]byte(validConfig))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		purpose Purpose
		ref     string
		want    string
	}{
		{PurposeGenesis, "genesis-wemix-test.json", "/data/genesis/genesis-wemix-test.json"},
		{PurposeConfigs, "config-wemix-test1.yml", "/data/configs/config-wemix-test1.yml"},
		{PurposeKeystore, "bp1/account.json", "/data/keystore/bp1/account.json"},
		{PurposeKeyrings, "preset-a", "/data/keys/preset-a"},
	}
	for _, tc := range cases {
		got, err := c.Resolve(tc.purpose, tc.ref)
		if err != nil {
			t.Fatalf("resolve %s/%s: %v", tc.purpose, tc.ref, err)
		}
		if got != tc.want {
			t.Errorf("resolve %s/%s = %q, want %q", tc.purpose, tc.ref, got, tc.want)
		}
	}
}

// TestResolve_RootChangeMovesEveryPath: swapping dataRoot moves every resolved
// path with it, which is the point of the environment file.
func TestResolve_RootChangeMovesEveryPath(t *testing.T) {
	other := strings.Replace(validConfig, "dataRoot: /data", "dataRoot: /srv/testroot", 1)
	c, err := ParseWorkspaceConfig([]byte(other))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Resolve(PurposeGenesis, "g.json")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/srv/testroot/genesis/g.json" {
		t.Fatalf("root change not reflected: %q", got)
	}
}

func TestBinaryPath_AliasApplied(t *testing.T) {
	c, err := ParseWorkspaceConfig([]byte(validConfig))
	if err != nil {
		t.Fatal(err)
	}
	// Aliased name maps to the file under paths.binaries.
	if got, _ := c.BinaryPath("gwemix"); got != "/data/bin/linux-amd64/gwemix" {
		t.Errorf("aliased binary = %q, want /data/bin/linux-amd64/gwemix", got)
	}
	// A name with no alias is used verbatim.
	if got, _ := c.BinaryPath("gwbft"); got != "/data/bin/gwbft" {
		t.Errorf("unaliased binary = %q, want /data/bin/gwbft", got)
	}
}

// TestResolve_RejectsUnportableRefs: an absolute, traversing, tilde, env-var, or
// empty reference is refused — the guard that keeps a portable reference portable.
func TestResolve_RejectsUnportableRefs(t *testing.T) {
	c, err := ParseWorkspaceConfig([]byte(validConfig))
	if err != nil {
		t.Fatal(err)
	}
	for _, ref := range []string{"/etc/passwd", "../escape", "a/../../b", "~/x", "$HOME/x", "${VAR}/x", "", "   "} {
		if _, err := c.Resolve(PurposeGenesis, ref); err == nil {
			t.Errorf("ref %q was accepted, want rejected", ref)
		}
	}
}

func TestValidate_Rejects(t *testing.T) {
	cases := map[string]func(string) string{
		"bad version":        func(s string) string { return strings.Replace(s, "version: 1", "version: 2", 1) },
		"missing dataRoot":   func(s string) string { return strings.Replace(s, "dataRoot: /data\n", "", 1) },
		"relative dataRoot":  func(s string) string { return strings.Replace(s, "dataRoot: /data", "dataRoot: data", 1) },
		"tilde dataRoot":     func(s string) string { return strings.Replace(s, "dataRoot: /data", "dataRoot: ~/data", 1) },
		"traversal path":     func(s string) string { return strings.Replace(s, "binaries: bin", "binaries: ../bin", 1) },
		"absolute path":      func(s string) string { return strings.Replace(s, "binaries: bin", "binaries: /bin", 1) },
		"unknown field":      func(s string) string { return s + "\nunknownField: x\n" },
		"unknown input mode": func(s string) string { return strings.Replace(s, "mode: generated", "mode: borrowed", 1) },
		"unknown chain mode": func(s string) string { return strings.Replace(s, "chain: fresh", "chain: recycled", 1) },
		"generated + preset": func(s string) string {
			return strings.Replace(s, "mode: generated", "mode: generated\n  preset: regression", 1)
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseWorkspaceConfig([]byte(mutate(validConfig))); err == nil {
				t.Fatalf("%s was accepted, want rejected", name)
			}
		})
	}
}

// TestInputs_PreparedNeedsPresetThatExists: prepared must name a preset, and the
// preset must be declared.
func TestInputs_PreparedNeedsPresetThatExists(t *testing.T) {
	prepared := `
version: 1
dataRoot: /data
paths: {binaries: bin, configs: configs, genesis: genesis, keystore: keystore, keyrings: keys, nodes: node, runtime: runtime, logs: logs}
control: {artifactRoot: ~/.chainbench}
inputs:
  mode: prepared
  preset: regression
execution:
  chain: fresh
presets:
  regression:
    genesis: srv://server-01/data/genesis/g.json
    keyring: srv://server-01/data/keys/r
    configs:
      default: srv://server-01/data/configs/c.toml
`
	if _, err := ParseWorkspaceConfig([]byte(prepared)); err != nil {
		t.Fatalf("valid prepared config rejected: %v", err)
	}
	// prepared with no preset -> error.
	noPreset := strings.Replace(prepared, "  preset: regression\n", "", 1)
	if _, err := ParseWorkspaceConfig([]byte(noPreset)); err == nil {
		t.Fatal("prepared with no preset was accepted")
	}
	// preset names a missing entry -> error.
	missing := strings.Replace(prepared, "preset: regression", "preset: nonesuch", 1)
	if _, err := ParseWorkspaceConfig([]byte(missing)); err == nil {
		t.Fatal("preset naming a missing entry was accepted")
	}
}

func TestArtifactRoot_LocalExpansion(t *testing.T) {
	// ~ expands to the local home.
	c, err := ParseWorkspaceConfig([]byte(validConfig))
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.ArtifactRoot()
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(got, "~") || !filepath.IsAbs(got) {
		t.Fatalf("artifactRoot ~ not expanded to an absolute local path: %q", got)
	}
	// A relative artifactRoot is relative to the config file's directory.
	rel := strings.Replace(validConfig, "artifactRoot: ~/.chainbench", "artifactRoot: out", 1)
	c2, err := ParseWorkspaceConfig([]byte(rel))
	if err != nil {
		t.Fatal(err)
	}
	c2.dir = "/tmp/cfgdir"
	if got, _ := c2.ArtifactRoot(); got != filepath.Join("/tmp/cfgdir", "out") {
		t.Fatalf("relative artifactRoot = %q, want under the config dir", got)
	}
}

// TestParseWorkspaceConfig_Sample parses the tracked sample with the real parser
// (acceptance A): the shipped proposal must at least parse and validate.
func TestParseWorkspaceConfig_Sample(t *testing.T) {
	c, err := LoadWorkspaceConfig(filepath.Join("..", "..", "workspace-config.sample.yaml"))
	if err != nil {
		t.Fatalf("the sample must parse with the real parser: %v", err)
	}
	if c.DataRoot != "/data" || c.Inputs.Mode != InputGenerated {
		t.Fatalf("sample parsed unexpectedly: dataRoot=%q mode=%q", c.DataRoot, c.Inputs.Mode)
	}
}
