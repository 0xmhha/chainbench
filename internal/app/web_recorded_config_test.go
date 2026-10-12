package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

// TestRecordedDeclarationsMatchWhatTheComposerRecorded: an SSH composition
// records its workspace config with control.binaries naming the directory the
// CLI shipped the accepted binary from. A later control compares the record
// with exactly that, so the network it composed is not refused as changed,
// while a config that differs in any other way still is.
func TestRecordedDeclarationsMatchWhatTheComposerRecorded(t *testing.T) {
	dir := t.TempDir()
	p := webChainPayload{ControlDir: dir, Target: resource.Inspection{Transport: "ssh"},
		Set: DeploymentDocument{DeploymentDocumentInput: deploymentTestSet()}, Config: DeploymentDocument{DeploymentDocumentInput: deploymentTestConfig()}}
	config, err := webRecordedWorkspaceConfig(p)
	if err != nil {
		t.Fatal(err)
	}
	wc, err := resource.ParseWorkspaceConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	if wc.Control.Binaries != filepath.Join(dir, "binaries") {
		t.Fatalf("control.binaries = %q", wc.Control.Binaries)
	}
	set, err := ExportDeploymentDocument(p.Set.DeploymentDocumentInput, "yaml")
	if err != nil {
		t.Fatal(err)
	}
	state := State{WorkspaceConfig: filepath.Join(dir, "workspace-config.yaml"), ServerSet: filepath.Join(dir, "server-set.yaml")}
	for path, b := range map[string][]byte{state.WorkspaceConfig: config, state.ServerSet: set} {
		if err = os.WriteFile(path, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err = recordedDeclarationsMatch(state, p); err != nil {
		t.Fatalf("the composer's own record was refused: %v", err)
	}
	if err = os.WriteFile(state.WorkspaceConfig, append(config, []byte("# edited\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if err = recordedDeclarationsMatch(state, p); err == nil {
		t.Fatal("a changed recorded config was accepted")
	}
}
