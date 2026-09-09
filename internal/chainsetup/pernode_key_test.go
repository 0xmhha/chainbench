package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins
)

// TestKeys_NodeTablePinnedKeyDrivesGenesis is S5's per-node key contract (cases
// a/b/c together): node1 pins a key and its address becomes the node's identity
// and a genesis validator (a); node2 names no key and is generated, and is a
// validator too (b); node3 is an endpoint — it gets a key but is not a validator
// (c). Genesis is key-driven, so pinning the key is what fixes the validator.
func TestKeys_NodeTablePinnedKeyDrivesGenesis(t *testing.T) {
	dir := t.TempDir()
	keysDir := filepath.Join(dir, "keys")

	// A fixed key so the test knows the address the pin must produce.
	const pinnedHex = "0x1111111111111111111111111111111111111111111111111111111111111111"
	pinnedKey, err := derive.ParsePrivateKey(pinnedHex)
	if err != nil {
		t.Fatal(err)
	}
	pinnedID, err := derive.Derive(pinnedKey, derive.WithBLS)
	if err != nil {
		t.Fatal(err)
	}
	wantAddr := pinnedID.Address

	ws, err := chainsetup.Open(dir, fixedClock())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ws.New(chainsetup.NewOpts{Chain: "stablenet", KeysDir: keysDir}); err != nil {
		t.Fatal(err)
	}
	topo := &node.Topology{Chain: "stablenet", Nodes: []node.Entry{
		{Index: 1, Role: "bp", Key: pinnedHex},
		{Index: 2, Role: "bp"},
		{Index: 3, Role: "en"},
	}}
	if _, err := ws.Allocate(chainsetup.AllocateOpts{Topology: topo}); err != nil {
		t.Fatalf("allocate: %v", err)
	}
	ctx := context.Background()
	if _, err := ws.Keys(ctx, chainsetup.KeysOpts{}); err != nil {
		t.Fatalf("keys: %v", err)
	}

	set, err := store.LoadPreset(keysDir)
	if err != nil {
		t.Fatalf("load preset: %v", err)
	}
	if len(set.Nodes) != 3 {
		t.Fatalf("preset has %d nodes, want 3", len(set.Nodes))
	}
	// (a) node1's identity is the pinned key.
	if got := set.Nodes[0].Address; !strings.EqualFold(got, wantAddr) {
		t.Errorf("node1 address = %s, want the pinned key's %s", got, wantAddr)
	}
	// (b) node2 was generated — a real, different address.
	if got := set.Nodes[1].Address; got == "" || strings.EqualFold(got, wantAddr) {
		t.Errorf("node2 address = %q, want a generated one distinct from node1", got)
	}
	// (c) node3 (en) has a key but is not a validator.
	if set.Nodes[2].Address == "" {
		t.Error("node3 (en) has no key, but a node needs one to run")
	}
	if len(set.Network.Validators) != 2 {
		t.Fatalf("validators = %v, want the two producers", set.Network.Validators)
	}
	if !strings.EqualFold(set.Network.Validators[0], wantAddr) {
		t.Errorf("first validator = %s, want node1's pinned %s", set.Network.Validators[0], wantAddr)
	}
	for _, v := range set.Network.Validators {
		if strings.EqualFold(v, set.Nodes[2].Address) {
			t.Errorf("node3 (en) address %s must not be a validator", v)
		}
	}

	// End to end: the composed genesis carries the pinned address.
	if _, err := ws.Genesis(ctx, chainsetup.GenesisOpts{}); err != nil {
		t.Fatalf("genesis: %v", err)
	}
	gen, err := os.ReadFile(filepath.Join(ws.State().Target.DataRoot, "genesis.json"))
	if err != nil {
		// The genesis path is derived from the target's data root; fall back to
		// the workspace dir if the root is the workspace.
		gen, err = os.ReadFile(filepath.Join(dir, "genesis.json"))
		if err != nil {
			t.Fatalf("read genesis: %v", err)
		}
	}
	if !strings.Contains(strings.ToLower(string(gen)), strings.ToLower(strings.TrimPrefix(wantAddr, "0x"))) {
		t.Errorf("composed genesis does not carry the pinned validator address %s", wantAddr)
	}
}
