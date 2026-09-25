package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/resource"
)

// TestRm_RemovesTheCompositionFromTheDataRoot: rm on a composition isolated by
// id leaves nothing of it on the data root — not its logs, and not the
// node/<id>, runtime/<id> and logs/<id> directories. It used to remove the
// datadirs, configs and genesis and keep the rest, and a data root that had
// seen a few hundred runs ran out of space.
func TestRm_RemovesTheCompositionFromTheDataRoot(t *testing.T) {
	dir, dataRoot := t.TempDir(), t.TempDir()
	wcPath := filepath.Join(dir, "workspace-config.yaml")
	must(t, os.WriteFile(wcPath, []byte(`version: 1
dataRoot: `+dataRoot+`
paths: {binaries: bin, configs: configs, genesis: genesis, keystore: keystore, keyrings: keys, nodes: node, runtime: runtime, logs: logs}
control: {artifactRoot: ~/.chainbench}
inputs: {mode: generated}
execution: {chain: fresh}
`), 0o644))

	ws, err := chainsetup.Open(dir, fixedClock())
	must(t, err)
	_, err = ws.New(chainsetup.NewOpts{
		Chain: "stablenet", KeysDir: filepath.Join("..", "..", "presets", "keys"),
		Target: resource.Spec{DataRoot: dataRoot}, WorkspaceConfigPath: wcPath,
	})
	must(t, err)
	_, err = ws.Allocate(chainsetup.AllocateOpts{BPCount: 2})
	must(t, err)
	ctx := context.Background()
	_, err = ws.Keys(ctx, chainsetup.KeysOpts{})
	must(t, err)
	_, err = ws.Genesis(ctx, chainsetup.GenesisOpts{})
	must(t, err)
	_, err = ws.Config(ctx)
	must(t, err)
	// What a launch leaves: a datadir and a log per node.
	for _, ns := range ws.State().Nodes {
		must(t, os.MkdirAll(ns.DataDir, 0o755))
		must(t, os.MkdirAll(filepath.Dir(ns.LogPath), 0o755))
		must(t, os.WriteFile(ns.LogPath, []byte("INFO started\n"), 0o644))
	}
	id := ws.State().CompositionID
	if id == "" {
		t.Fatal("a workspace-config composition has no id, so there is nothing to isolate")
	}

	if _, err := ws.Rm(ctx); err != nil {
		t.Fatalf("rm: %v", err)
	}
	for _, sub := range []string{"node", "runtime", "logs"} {
		p := filepath.Join(dataRoot, sub, id)
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("rm left %s behind", p)
		}
	}
}
