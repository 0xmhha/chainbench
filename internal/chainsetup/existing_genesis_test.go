package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
)

// TestGenesis_ExistingIsUsedVerbatim is W4's finished-genesis contract: a genesis
// named by an "existing" reference is written to the target verbatim, not built
// from a template. This is what lets a prepared bundle (existing genesis +
// preset keyring + per-node configs) drive a regression run.
func TestGenesis_ExistingIsUsedVerbatim(t *testing.T) {
	dir := t.TempDir()
	presetDir := filepath.Join("..", "..", "keys", "preset")
	// A distinctive finished genesis (valid JSON, not what the builder would
	// make) that still names the validators the two-producer key set provides —
	// a wbft genesis without them is refused, and rightly, since it cannot sign.
	preset, err := store.LoadPreset(presetDir)
	if err != nil {
		t.Fatal(err)
	}
	net := preset.NetworkFor(2)
	extra, err := wbft.ExtraData(net.Validators, net.BLSKeys)
	if err != nil {
		t.Fatal(err)
	}
	finished := `{"config":{"chainId":424242},"note":"prepared-regression","alloc":{},"extraData":"` + extra + `"}`
	genPath := filepath.Join(dir, "prepared-genesis.json")
	if err := os.WriteFile(genPath, []byte(finished), 0o644); err != nil {
		t.Fatal(err)
	}

	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", KeysDir: presetDir}); err != nil {
		t.Fatal(err)
	}
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp"}, {Index: 2, Role: "bp"},
	}}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Topology: topo}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	ctx := context.Background()
	if _, err := ws.Keys(ctx, chainsetup.KeysOpts{}); err != nil {
		t.Fatalf("keys: %v", err)
	}
	if _, err := ws.Genesis(ctx, chainsetup.GenesisOpts{Existing: genPath}); err != nil {
		t.Fatalf("genesis (existing): %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "genesis.json"))
	if err != nil {
		t.Fatalf("read genesis: %v", err)
	}
	if string(got) != finished {
		t.Fatalf("genesis is not the finished file verbatim:\n%s", got)
	}
}

// TestGenesis_ExistingRejectsInvalidJSON: a finished genesis that is not valid
// JSON fails the step rather than being written and failing at node init.
func TestGenesis_ExistingRejectsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	genPath := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(genPath, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", KeysDir: filepath.Join("..", "..", "keys", "preset")}); err != nil {
		t.Fatal(err)
	}
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{{Index: 1, Role: "bp"}, {Index: 2, Role: "bp"}}}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Topology: topo}); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := ws.Keys(ctx, chainsetup.KeysOpts{}); err != nil {
		t.Fatal(err)
	}
	if _, err := ws.Genesis(ctx, chainsetup.GenesisOpts{Existing: genPath}); err == nil {
		t.Fatal("an invalid-JSON existing genesis was accepted")
	}
}
