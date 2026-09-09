package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins
)

// TestConfig_PinnedFileIsUsedVerbatim is S4's per-node config contract: a node
// whose topology entry names a config file gets that file at its config path,
// byte for byte, while its neighbours are rendered as usual. The write goes
// through the same path (and readback checksum) as a rendered config, so a
// pinned file that fails to copy is caught here, not at boot.
func TestConfig_PinnedFileIsUsedVerbatim(t *testing.T) {
	dir := t.TempDir()
	const pinned = "# pinned by the test\n[Node]\nDataDir = \"/somewhere/of/its/own\"\n"
	pinnedPath := filepath.Join(dir, "node1-pinned.toml")
	if err := os.WriteFile(pinnedPath, []byte(pinned), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{
		Chain: "stablenet", KeysDir: filepath.Join("..", "..", "keys", "preset"),
	}); err != nil {
		t.Fatal(err)
	}
	// node1 pins a config file; node2 renders normally.
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp", Config: pinnedPath},
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

	// node1's config is the pinned file, verbatim.
	got, err := os.ReadFile(filepath.Join(dir, "config_node1.toml"))
	if err != nil {
		t.Fatalf("read node1 config: %v", err)
	}
	if string(got) != pinned {
		t.Fatalf("node1 config is not the pinned file verbatim:\n%s", got)
	}
	// node2 was rendered — it is a real config, not the pinned one.
	got2, err := os.ReadFile(filepath.Join(dir, "config_node2.toml"))
	if err != nil {
		t.Fatalf("read node2 config: %v", err)
	}
	if string(got2) == pinned {
		t.Fatal("node2 must be rendered, not the pinned file")
	}
	if len(got2) == 0 {
		t.Fatal("node2 config is empty")
	}
}

// TestConfig_PinnedFileMissingIsReported: a config path that does not exist
// fails the config step with a message naming the node and the path, rather
// than launching a node with no config.
func TestConfig_PinnedFileMissingIsReported(t *testing.T) {
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{
		Chain: "stablenet", KeysDir: filepath.Join("..", "..", "keys", "preset"),
	}); err != nil {
		t.Fatal(err)
	}
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp", Config: filepath.Join(dir, "does-not-exist.toml")},
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
	if _, err := ws.Config(ctx); err == nil {
		t.Fatal("a missing pinned config was accepted")
	}
}
