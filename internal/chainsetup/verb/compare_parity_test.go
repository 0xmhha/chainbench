package verb

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// A run composes through ChainUpComparing and `chain up` through composeFrom.
// These hold the run to what chain up is held to; each failed before the run
// path went through planUp.

// TestChainUpComparing_RefusesAnInlineKeyBeforeRecordingIt: a node table that
// carries a private key inline is refused before the workspace is touched, so
// the key never reaches chain-record.json.
func TestChainUpComparing_RefusesAnInlineKeyBeforeRecordingIt(t *testing.T) {
	dir := t.TempDir()
	in := chainsetup.ChainUpIn{
		DataDir: dir, Chain: "stablenet", Binary: "/opt/gstable", Stage: chainsetup.UpStart,
		Topology: &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
			{Index: 1, Role: "bp", Key: "0x1111111111111111111111111111111111111111111111111111111111111111"},
		}},
	}
	_, err := ChainUpComparing(context.Background(), chainsetup.Deps{}, CompareIn{Up: in})
	if err == nil || !strings.Contains(err.Error(), "inline") {
		t.Fatalf("an inline key was not refused: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(dir, "chain-record.json")); !os.IsNotExist(serr) {
		t.Error("the refused request was recorded in the workspace anyway")
	}
}

// TestChainUpComparing_HonorsExecutionChainAttach: a workspace-config that says
// attach means this run composes nothing — the setting a run used to ignore.
func TestChainUpComparing_HonorsExecutionChainAttach(t *testing.T) {
	dir := t.TempDir()
	wcPath := filepath.Join(dir, "workspace-config.yaml")
	if err := os.WriteFile(wcPath, []byte(`version: 1
dataRoot: /data/chainbench
paths: {binaries: bin, configs: configs, genesis: genesis, keystore: keystore, keyrings: keys, nodes: node, runtime: runtime, logs: logs}
control: {artifactRoot: ~/.chainbench}
inputs: {mode: generated}
execution: {chain: attach}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	in := chainsetup.ChainUpIn{
		DataDir: filepath.Join(dir, "ws"), Chain: "stablenet", Binary: "/opt/gstable",
		Stage: chainsetup.UpStart, BPCount: 1, WorkspaceConfigPath: wcPath,
	}
	_, err := ChainUpComparing(context.Background(), chainsetup.Deps{}, CompareIn{Up: in})
	if err == nil || !strings.Contains(err.Error(), "execution.chain=attach") {
		t.Fatalf("execution.chain=attach was ignored: %v", err)
	}
}
