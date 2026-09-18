package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// rawBlueprint is a declaration that carries its own keys: no preset directory
// is named anywhere in it, which is the whole point of the raw path.
//
// TEST MATERIAL ONLY — the keys are a repeated byte, which is exactly what a
// real key must never be.
func rawBlueprint(t *testing.T) string {
	t.Helper()
	doc := `chain: wbft
nodes:
  - {name: bp1, role: bp, nodekey: {hex: "0x` + strings.Repeat("11", 32) + `"}}
  - {name: bp2, role: bp, nodekey: {hex: "0x` + strings.Repeat("22", 32) + `"}}
  - {name: en1, role: en, nodekey: {hex: "0x` + strings.Repeat("33", 32) + `"}}
`
	path := filepath.Join(t.TempDir(), "network.yaml")
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestBlueprint_ComposesWithNoPresetDirectory is N3's gate at the composition
// level: place and keys both run, and the key set is materialised from the
// document rather than loaded from a preset that does not exist.
func TestBlueprint_ComposesWithNoPresetDirectory(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	d := chainsetup.Deps{Clock: fixedClock()}
	keysDir := filepath.Join(dir, "keys")

	if _, err := chainsetup.NetNew(ctx, d, chainsetup.NetNewIn{
		DataDir: dir, Chain: "wbft", KeysDir: keysDir,
	}); err != nil {
		t.Fatalf("new: %v", err)
	}
	// The directory does not exist, which under the preset source is a failure
	// and under this one is simply where the ring will land.
	if _, err := os.Stat(keysDir); !os.IsNotExist(err) {
		t.Fatalf("the key directory should not exist yet: %v", err)
	}

	bp := rawBlueprint(t)
	if _, err := chainsetup.NetAllocate(ctx, d, chainsetup.NetAllocateIn{
		DataDir: dir, BlueprintPath: bp,
	}); err != nil {
		t.Fatalf("allocate from a blueprint: %v", err)
	}
	if _, err := chainsetup.NetKeys(ctx, d, chainsetup.NetKeysIn{
		DataDir: dir, BlueprintPath: bp,
	}); err != nil {
		t.Fatalf("keys from a blueprint: %v", err)
	}

	st := stateOf(t, dir, d)
	if len(st.Nodes) != 3 {
		t.Fatalf("composed %d nodes, want 3", len(st.Nodes))
	}
	want := []string{"bp", "bp", "en"}
	for i, n := range st.Nodes {
		if n.Role != want[i] {
			t.Errorf("node%d role = %q, want %q from the declaration", i+1, n.Role, want[i])
		}
	}
	// The ring landed, and it holds the declared keys rather than fresh ones.
	got, err := os.ReadFile(filepath.Join(keysDir, "node1", "nodekey"))
	if err != nil {
		t.Fatalf("the declared ring was not written: %v", err)
	}
	if strings.TrimSpace(string(got)) != strings.Repeat("11", 32) {
		t.Errorf("node1 key = %s, want the declared one", got)
	}

	// Genesis and config run off that ring, which is the proof that nothing
	// downstream had to learn a second way to be given keys.
	if _, err := chainsetup.NetGenesis(ctx, d, chainsetup.NetGenesisIn{DataDir: dir}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	if _, err := chainsetup.NetConfig(ctx, d, chainsetup.NetConfigIn{DataDir: dir}); err != nil {
		t.Fatalf("config: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "genesis.json")); err != nil {
		t.Errorf("no genesis was built: %v", err)
	}
}

// TestBlueprint_RefusesTwoDescriptionsOfTheLayout: a blueprint and a topology
// both say what the network is. Picking one silently would leave the other's
// author reading a network that is not theirs.
func TestBlueprint_RefusesTwoDescriptionsOfTheLayout(t *testing.T) {
	dir := t.TempDir()
	d := chainsetup.Deps{Clock: fixedClock()}
	if _, err := chainsetup.NetNew(context.Background(), d, chainsetup.NetNewIn{DataDir: dir, Chain: "wbft"}); err != nil {
		t.Fatalf("new: %v", err)
	}
	topo := filepath.Join(t.TempDir(), "topology.yaml")
	if err := os.WriteFile(topo, []byte("chain: wbft\nnodes:\n  - {index: 1, role: bp}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := chainsetup.NetAllocate(context.Background(), d, chainsetup.NetAllocateIn{
		DataDir: dir, BlueprintPath: rawBlueprint(t), TopologyPath: topo,
	})
	if err == nil {
		t.Fatal("a blueprint and a topology were both accepted")
	}
	if !strings.Contains(err.Error(), "give one") {
		t.Errorf("error %q should say to give one", err)
	}
}

// TestBlueprint_RefusesAFieldItCannotHonour: a declared value that the
// allocation cannot act on is refused by name. Dropping it would be the
// failure this whole track exists to end — a network that comes up looking
// right and running something the document does not describe.
func TestBlueprint_RefusesAFieldItCannotHonour(t *testing.T) {
	dir := t.TempDir()
	d := chainsetup.Deps{Clock: fixedClock()}
	if _, err := chainsetup.NetNew(context.Background(), d, chainsetup.NetNewIn{DataDir: dir, Chain: "wbft"}); err != nil {
		t.Fatalf("new: %v", err)
	}
	path := filepath.Join(t.TempDir(), "network.yaml")
	doc := "chain: wbft\nnodes:\n  - {name: bp1, role: bp, server: srv2, nodekey: {hex: \"0x" + strings.Repeat("11", 32) + "\"}}\n"
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := chainsetup.NetAllocate(context.Background(), d, chainsetup.NetAllocateIn{DataDir: dir, BlueprintPath: path})
	if err == nil {
		t.Fatal("a per-node server was silently ignored")
	}
	if !strings.Contains(err.Error(), "srv2") {
		t.Errorf("error %q should name the field it could not honour", err)
	}
}
