package upgrade_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/resource"
)

// serverNamed is a reference to one server set entry.
func serverNamed(name string) resource.ServerRef { return resource.ServerRef{Name: name} }

// writeTargetConfig puts an environment file on disk and returns its path.
func writeTargetConfig(t *testing.T, dataRoot string) string {
	t.Helper()
	body := "version: 1\ndataRoot: " + dataRoot + "\n" +
		"paths:\n  binaries: bin\n  configs: configs\n  genesis: genesis\n" +
		"  keystore: keystore\n  keyrings: keys\n  nodes: node\n  runtime: runtime\n  logs: logs\n" +
		"control:\n  artifactRoot: " + dataRoot + "\ninputs:\n  mode: generated\nexecution:\n  chain: fresh\n"
	p := filepath.Join(t.TempDir(), "workspace-config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestTarget_LocalWithAnEnvironmentFileStillTakesItsDataRoot.
//
// An environment file says where files are, and that is as true of this machine
// as of any other. A local handoff given one used to ignore it and compose
// under the workspace directory instead.
func TestTarget_LocalWithAnEnvironmentFileStillTakesItsDataRoot(t *testing.T) {
	cfg := writeTargetConfig(t, "/data/net1")
	in := upgrade.HandoffInputs{DataDir: "/tmp/ws"}

	wc, err := upgrade.Target{WorkspaceConfigPath: cfg}.Apply(&in, 5)
	if err != nil {
		t.Fatal(err)
	}
	if wc == nil {
		t.Fatal("the environment file was not returned, so nothing can place a binary name with it")
	}
	if in.DataDir != "/data/net1" {
		t.Errorf("data dir = %q, want the environment file's %q", in.DataDir, "/data/net1")
	}
	// Local stays local: no placement, no per-node machines, no remote stores.
	if in.Placement != nil || in.Machine != nil || in.Files != nil || in.Driver != nil {
		t.Error("a local target resolved machines it does not have")
	}
}

// TestTarget_LocalWithNoEnvironmentFileChangesNothing keeps the shape a handoff
// has always had when nobody said where to run it.
func TestTarget_LocalWithNoEnvironmentFileChangesNothing(t *testing.T) {
	in := upgrade.HandoffInputs{DataDir: "/tmp/ws"}
	wc, err := upgrade.Target{}.Apply(&in, 5)
	if err != nil {
		t.Fatal(err)
	}
	if wc != nil || in.DataDir != "/tmp/ws" {
		t.Errorf("a bare target changed the run: wc=%v dataDir=%q", wc, in.DataDir)
	}
}

// TestTarget_AServerNeedsAnEnvironmentFile: a remote machine cannot resolve
// without knowing where its data lives, and saying so names the missing file
// instead of failing further down as a resolution error.
func TestTarget_AServerNeedsAnEnvironmentFile(t *testing.T) {
	in := upgrade.HandoffInputs{}
	_, err := upgrade.Target{Server: serverNamed("alpha")}.Apply(&in, 5)
	if err == nil {
		t.Fatal("a server with no environment file was accepted")
	}
	if want := "workspace-config is required"; !strings.Contains(err.Error(), want) {
		t.Errorf("the refusal should name what is missing: %v", err)
	}
}
