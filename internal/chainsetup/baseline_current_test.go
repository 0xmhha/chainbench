package chainsetup

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// baselineWorkspace builds a workspace whose composed inputs are real files on a
// local target, with a workspace-config beside which the baseline lives.
func baselineWorkspace(t *testing.T) (*Workspace, string, string) {
	t.Helper()
	dir := t.TempDir()
	dataRoot := t.TempDir()
	wcPath := filepath.Join(dir, "workspace-config.yaml")
	body := "version: 1\ndataRoot: " + dataRoot + "\n" +
		"paths: {binaries: bin, configs: configs, genesis: genesis, keystore: keystore, keyrings: keys, nodes: node, runtime: runtime, logs: logs}\n" +
		"control: {artifactRoot: ~/.chainbench}\ninputs: {mode: generated}\nexecution: {chain: fresh}\n"
	if err := os.WriteFile(wcPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}

	genesisPath := filepath.Join(dataRoot, "runtime", "abc123", "genesis.json")
	cfgPath := filepath.Join(dataRoot, "runtime", "abc123", "configs", "node1.toml")
	files := filestore.Local{}
	ctx := context.Background()
	if err := files.Write(ctx, genesisPath, []byte(`{"config":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := files.Write(ctx, cfgPath, []byte("[Node]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	w, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.SetEnv(os.Getenv)
	w.state.Chain = "stablenet"
	w.state.WorkspaceConfig = wcPath
	w.state.Target = resource.Spec{DataRoot: dataRoot}
	w.state.GenesisPath = genesisPath
	w.state.Nodes = []node.Record{{Index: 1, Label: "node1", ConfigPath: cfgPath}}
	return w, cfgPath, genesisPath
}

// TestBaseline_DetectsAServerSideEdit is MON-016's first condition, and the case
// the original live check could not see: it changed a config by re-running the
// config step, which rewrites the recorded hash too. Here nothing is recomposed
// — only the file on the target changes — which is exactly how a prepared input
// drifts under a regression environment.
func TestBaseline_DetectsAServerSideEdit(t *testing.T) {
	w, cfgPath, genesisPath := baselineWorkspace(t)
	ctx := context.Background()

	obs, err := w.ObserveBaseline(ctx)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	path := resource.BaselinePathFor(w.state.WorkspaceConfig)
	if err := resource.SaveBaseline(path, resource.Baseline{
		Genesis: obs.Genesis, Configs: obs.Configs, Validators: obs.Validators,
	}); err != nil {
		t.Fatal(err)
	}

	// Approved, and nothing has moved yet.
	check, err := w.CheckBaseline(ctx)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if !check.Approved || !check.Match {
		t.Fatalf("a freshly approved baseline must match: %+v", check)
	}

	// Someone edits the config ON THE TARGET. The workspace is not touched, so
	// its recorded hash still describes what it composed.
	if err := os.WriteFile(cfgPath, []byte("[Node]\nedited = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	check, err = w.CheckBaseline(ctx)
	if err != nil {
		t.Fatalf("check after edit: %v", err)
	}
	if check.Match {
		t.Fatal("an edit to the config on the target was reported as a match")
	}
	if !strings.Contains(strings.Join(check.Diffs, " "), "node1 config") {
		t.Fatalf("the drift should name the config: %v", check.Diffs)
	}

	// And a deleted input is a failure to check, not a match.
	if err := os.Remove(genesisPath); err != nil {
		t.Fatal(err)
	}
	if _, err := w.CheckBaseline(ctx); err == nil {
		t.Fatal("a genesis that cannot be read must not check clean")
	}
}

// TestBaseline_CheckChangesNothing: the check reads; it must not rewrite the
// approval or the environment.
func TestBaseline_CheckChangesNothing(t *testing.T) {
	w, cfgPath, _ := baselineWorkspace(t)
	ctx := context.Background()
	obs, err := w.ObserveBaseline(ctx)
	if err != nil {
		t.Fatal(err)
	}
	path := resource.BaselinePathFor(w.state.WorkspaceConfig)
	if err := resource.SaveBaseline(path, resource.Baseline{Genesis: obs.Genesis, Configs: obs.Configs}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cfgBefore, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(cfgPath, []byte("[Node]\ndrifted = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := w.CheckBaseline(ctx); err != nil {
		t.Fatalf("check: %v", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("the check rewrote the approved baseline")
	}
	// The check must not repair the drift either.
	if now, _ := os.ReadFile(cfgPath); string(now) == string(cfgBefore) {
		t.Fatal("the check rewrote the target's config")
	}
}
