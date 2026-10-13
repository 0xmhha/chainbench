package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// TestReconcileCountsDeployedFilesAsPresent: a deployment interrupted after
// its deploy stage has written node configuration to the target, though no
// node was initialised yet. Those files are this job's, so the network is not
// absent and its resources must stay claimed against other workspaces.
func TestReconcileCountsDeployedFilesAsPresent(t *testing.T) {
	e, p := webRunPlanFixture(t, `{"bp":4}`)
	root := t.TempDir()
	config := filepath.Join(root, "configs", "node1.toml")
	if err := os.MkdirAll(filepath.Dir(config), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config, []byte("[Node]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	state := State{Chain: "stablenet", Target: resource.Spec{DataRoot: root}, Nodes: []node.Record{
		{Index: 1, Label: "node1", Role: "bp", Host: "127.0.0.1", DataDir: filepath.Join(root, "nodes", "node1"), ConfigPath: config}}}
	raw, _ := json.Marshal(state)
	if err := os.MkdirAll(p.ControlDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.ControlDir, "chain-record.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(p)
	got, err := e.Reconcile(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, WebPreparedJob{Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if got.Outcome == "absent" || len(got.Nodes) != 1 || got.Nodes[0].Data != "present" {
		t.Fatalf("reconciliation = %+v, want the deployed configuration counted as present", got)
	}
}
