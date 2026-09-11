package testengine

import (
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// TestRingFor_RegistersEntriesThatCanSign is a regression test for a failure the
// suite did not catch and a live run did.
//
// ringFor builds the ring a spec signs with -- "from": "dev1", sendTx's "key"
// argument, every harness-signed transaction. It loads a preset and registers
// the entries into a KeySet, and a KeySet refuses an all-zero key. When the ring
// index stopped carrying private keys, this call kept reading identities only,
// so every run died at startup with:
//
//	keyring: register identity node1: keyring: add "node1": invalid private key: all zero
//
// Nothing in the package covered it, because the only tests that reached ringFor
// passed a directory with no ring at all (the "attaching to someone else's
// network" case), which returns early. This one gives it a real ring.
func TestRingFor_RegistersEntriesThatCanSign(t *testing.T) {
	dir := t.TempDir()
	if _, err := store.Generate(store.GenerateOpts{
		Out: dir, Nodes: 2, Password: "x",
	}, nil); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	ring, err := ringFor(dir)
	if err != nil {
		t.Fatalf("ringFor: %v", err)
	}
	if ring == nil {
		t.Fatal("a directory holding a ring produced none")
	}
	for _, label := range []keyring.Label{"node1", "node2"} {
		e, ok := ring.Get(label)
		if !ok {
			t.Fatalf("%s is not in the ring", label)
		}
		if e.Nodekey == (derive.PrivateKey{}) {
			t.Errorf("%s was registered without its key, so nothing can sign as it", label)
		}
		if e.Address == "" {
			t.Errorf("%s was registered without an address", label)
		}
	}
}

// TestRingFor_NoRingIsNotAnError keeps the case that WAS covered: attaching to a
// network somebody else composed carries the default keys directory in even when
// nothing is there, and an absent ring is a valid state rather than a failure.
func TestRingFor_NoRingIsNotAnError(t *testing.T) {
	ring, err := ringFor(filepath.Join(t.TempDir(), "nothing-here"))
	if err != nil {
		t.Fatalf("an absent ring must not be an error: %v", err)
	}
	if ring != nil {
		t.Fatal("an absent ring produced a ring")
	}
}
