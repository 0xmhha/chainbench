package chainsetup

import (
	"os"
	"path/filepath"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // the key source needs the chain's plugin
	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestKeySource_ATableThatNamesAKeyIsTheDeclaredWay.
//
// The third of the keys stage's leaves, and the only one a request cannot be
// read for: a composition that names no source and a node table that pins a key
// take the table's identities, not a preset's. Reading the request alone called
// that a preset, and two runs of the same request then produced different
// validator addresses with nothing in the record to say why.
//
// manager_test.go walks the other two leaves through a whole composition. This
// one is asked of the chooser directly, because a node table with a key file in
// it is a world to build and the question is only which way it settles.
func TestKeySource_ATableThatNamesAKeyIsTheDeclaredWay(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "node1.key")
	const pinned = "0x1111111111111111111111111111111111111111111111111111111111111111"
	if err := os.WriteFile(keyFile, []byte(pinned), 0o600); err != nil {
		t.Fatal(err)
	}

	ws, err := Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(NewOpts{Chain: "stablenet", KeysDir: filepath.Join(dir, "keys")}); err != nil {
		t.Fatal(err)
	}
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp", Key: keyFile},
		{Index: 2, Role: "bp"},
	}}
	if _, err := ws.Allocate(AllocateOpts{Topology: topo}); err != nil {
		t.Fatalf("allocate: %v", err)
	}

	// No source named: the request alone would say preset.
	_, way, _, err := ws.keySource(t.Context(), KeysOpts{})
	if err != nil {
		t.Fatalf("keySource: %v", err)
	}
	if way != wayDeclared {
		t.Errorf("a table that pins node1's key settled on %q, want %q", way, wayDeclared)
	}
	if got := newEnsuringKeys(&Manager{}).leaves[way].Name(); got != nameChainEnsureKeysFromBlueprint {
		t.Errorf("the declared way is carried out by %s, want %s", got, nameChainEnsureKeysFromBlueprint)
	}
}
