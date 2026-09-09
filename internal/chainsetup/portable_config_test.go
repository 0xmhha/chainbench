package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
)

// TestConfig_PortableRefResolvesUnderWorkspaceConfig is W2's consumption
// contract for the config site: a node config named by a portable relative
// reference is read from the target under the workspace-config's data root and
// configs directory — not as a local path relative to the caller. The same
// reference would resolve under a different root if the workspace-config named
// one, which is the point of the environment file.
func TestConfig_PortableRefResolvesUnderWorkspaceConfig(t *testing.T) {
	dir := t.TempDir()      // local control directory
	dataRoot := t.TempDir() // the target's data root
	const prepared = "# prepared on the target\n[Node]\nDataDir = \"x\"\n"

	// Place the prepared config where a portable ref resolves: dataRoot/configs.
	if err := os.MkdirAll(filepath.Join(dataRoot, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "configs", "prepared.toml"), []byte(prepared), 0o644); err != nil {
		t.Fatal(err)
	}
	// A workspace-config whose data root is the target's.
	wcPath := filepath.Join(dir, "workspace-config.yaml")
	if err := os.WriteFile(wcPath, []byte(`version: 1
dataRoot: `+dataRoot+`
paths: {binaries: bin, configs: configs, genesis: genesis, keystore: keystore, keyrings: keys, nodes: node, runtime: runtime, logs: logs}
control: {artifactRoot: ~/.chainbench}
inputs: {mode: generated}
execution: {chain: fresh}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{
		Chain: "stablenet", KeysDir: filepath.Join("..", "..", "keys", "preset"),
		Target:              resource.Spec{DataRoot: dataRoot},
		WorkspaceConfigPath: wcPath,
	}); err != nil {
		t.Fatal(err)
	}
	// node1 names a portable config ref; node2 renders normally.
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp", Config: "prepared.toml"},
		{Index: 2, Role: "bp"},
	}}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Topology: topo}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	ctx := context.Background()
	if _, err := ws.Keys(ctx, chainsetup.KeysOpts{}); err != nil {
		t.Fatalf("keys: %v", err)
	}
	if _, err := ws.Genesis(ctx, chainsetup.GenesisOpts{}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	if _, err := ws.Config(ctx); err != nil {
		t.Fatalf("config: %v", err)
	}

	// node1's written config is the prepared file read from dataRoot/configs.
	got, err := os.ReadFile(filepath.Join(dataRoot, "config_node1.toml"))
	if err != nil {
		t.Fatalf("read node1 config: %v", err)
	}
	if string(got) != prepared {
		t.Fatalf("node1 config was not the portable prepared file:\n%s", got)
	}
}
