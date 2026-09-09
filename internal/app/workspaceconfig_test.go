package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

const wcYAML = `
version: 1
dataRoot: /data/chainbench
paths:
  binaries: bin
  configs: configs
  genesis: genesis
  keystore: keystore
  keyrings: keys
  nodes: node
  runtime: runtime
  logs: logs
control:
  artifactRoot: ~/.chainbench
inputs:
  mode: generated
execution:
  chain: fresh
`

// writeWC writes a workspace-config fixture and returns its path.
func writeWC(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "workspace-config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestWithWorkspaceConfig_FoldsDataRoot proves the shared helper both surfaces
// call resolves the target's data root from the config file.
func TestWithWorkspaceConfig_FoldsDataRoot(t *testing.T) {
	got, err := WithWorkspaceConfig(resource.Spec{}, writeWC(t, wcYAML))
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	if got.DataRoot != "/data/chainbench" {
		t.Fatalf("dataRoot = %q, want /data/chainbench", got.DataRoot)
	}
}

// TestWithWorkspaceConfig_EmptyPathLeavesTargetUnchanged: no config, no change.
func TestWithWorkspaceConfig_EmptyPathLeavesTargetUnchanged(t *testing.T) {
	in := resource.Spec{DataRoot: "/somewhere", Server: "host"}
	got, err := WithWorkspaceConfig(in, "")
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if got != in {
		t.Fatalf("target changed: %+v", got)
	}
}

// TestWithWorkspaceConfig_ConflictingDataRootFails: the config is the single
// owner of the data root, so a target naming a different one is an error.
func TestWithWorkspaceConfig_ConflictingDataRootFails(t *testing.T) {
	_, err := WithWorkspaceConfig(resource.Spec{DataRoot: "/other"}, writeWC(t, wcYAML))
	if err == nil {
		t.Fatal("expected a data-root conflict error, got nil")
	}
	if !strings.Contains(err.Error(), "data root conflict") {
		t.Fatalf("error = %q, want a data-root conflict", err)
	}
}

// TestWithWorkspaceConfig_MatchingDataRootIsNoConflict: the same root on both
// sides is not a conflict — the run path may legitimately carry it.
func TestWithWorkspaceConfig_MatchingDataRootIsNoConflict(t *testing.T) {
	got, err := WithWorkspaceConfig(resource.Spec{DataRoot: "/data/chainbench"}, writeWC(t, wcYAML))
	if err != nil {
		t.Fatalf("matching: %v", err)
	}
	if got.DataRoot != "/data/chainbench" {
		t.Fatalf("dataRoot = %q", got.DataRoot)
	}
}

// TestTargetForWorkspaceConfig_EmptyPathIsZeroTarget: the MCP up path passes no
// target of its own; an empty config yields the zero target.
func TestTargetForWorkspaceConfig_EmptyPathIsZeroTarget(t *testing.T) {
	got, err := TargetForWorkspaceConfig("")
	if err != nil {
		t.Fatalf("empty: %v", err)
	}
	if (got != resource.Spec{}) {
		t.Fatalf("want zero target, got %+v", got)
	}
}
