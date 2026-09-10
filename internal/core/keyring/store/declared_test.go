package store_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// declaredSet builds a small ring the way a declaration would: keys stated,
// identities derived, nothing generated.
func declaredSet(t *testing.T, n int) keyring.Preset {
	t.Helper()
	var set keyring.Preset
	for i := 1; i <= n; i++ {
		k, err := derive.ParsePrivateKey(strings.Repeat(string(rune('0'+i)), 64))
		if err != nil {
			t.Fatalf("key %d: %v", i, err)
		}
		id, err := derive.Derive(k, derive.AccountOnly)
		if err != nil {
			t.Fatalf("derive %d: %v", i, err)
		}
		set.Nodes = append(set.Nodes, keyring.Entry{
			Label: keyring.Label("node" + string(rune('0'+i))), Index: i, Nodekey: k, Identity: id,
		})
		set.Network.Validators = append(set.Network.Validators, id.Address)
	}
	return set
}

// TestDeclaredKeys_WritesWhatWasDeclared: this source derives nothing, so the
// identities that land are exactly the ones the document stated. If it
// generated anything, a blueprint would describe one network and compose
// another.
func TestDeclaredKeys_WritesWhatWasDeclared(t *testing.T) {
	dir := t.TempDir()
	want := declaredSet(t, 3)

	got, err := store.DeclaredKeys{Path: dir, Set: want}.Ensure(context.Background(), 3)
	if err != nil {
		t.Fatalf("ensure: %v", err)
	}
	if len(got.Nodes) != 3 {
		t.Fatalf("loaded %d entries, want 3", len(got.Nodes))
	}
	for i, e := range got.Nodes {
		if e.Address != want.Nodes[i].Address || e.PublicKey != want.Nodes[i].PublicKey {
			t.Errorf("node%d came back as %s/%s, declared %s/%s",
				i+1, e.Address, e.PublicKey, want.Nodes[i].Address, want.Nodes[i].PublicKey)
		}
		if e.Nodekey.Hex() != want.Nodes[i].Nodekey.Hex() {
			t.Errorf("node%d key was not the declared one", i+1)
		}
	}
	// The shared password file is what a node unlocks with at launch
	// (--password). Its absence is why the first version of this source
	// composed cleanly, reported success at every step, and then had every
	// node die with "Failed to read password file".
	if _, err := os.Stat(filepath.Join(dir, "password")); err != nil {
		t.Errorf("no shared password file, so no node can unlock its account: %v", err)
	}
	// A node has to be able to unlock its account, so the keystore is part of
	// the set rather than an extra a caller remembers to write.
	for i := 1; i <= 3; i++ {
		nodeDir := filepath.Join(dir, "node"+string(rune('0'+i)))
		for _, name := range []string{"nodekey", "address", "pubkey"} {
			if _, err := os.Stat(filepath.Join(nodeDir, name)); err != nil {
				t.Errorf("node%d is missing %s: %v", i, name, err)
			}
		}
		ents, err := os.ReadDir(filepath.Join(nodeDir, "keystore"))
		if err != nil || len(ents) == 0 {
			t.Errorf("node%d has no keystore to unlock: %v", i, err)
		}
	}
}

// TestDeclaredKeys_ReusesWhatIsAlreadyThere is the rule every source on this
// boundary keeps: re-running must not replace identities a genesis and a
// datadir already refer to, because the keys behind them cannot be recovered.
func TestDeclaredKeys_ReusesWhatIsAlreadyThere(t *testing.T) {
	dir := t.TempDir()
	first := declaredSet(t, 2)
	if _, err := (store.DeclaredKeys{Path: dir, Set: first}).Ensure(context.Background(), 2); err != nil {
		t.Fatalf("first ensure: %v", err)
	}

	// A second call declaring different keys must not overwrite the first set.
	second := declaredSet(t, 2)
	second.Nodes[0].Nodekey = first.Nodes[1].Nodekey
	second.Nodes[0].Identity = first.Nodes[1].Identity

	got, err := store.DeclaredKeys{Path: dir, Set: second}.Ensure(context.Background(), 2)
	if err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	if got.Nodes[0].Address != first.Nodes[0].Address {
		t.Errorf("node1 became %s; the set on disk must win, or a running chain loses its keys",
			got.Nodes[0].Address)
	}
}

// TestDeclaredKeys_RefusesTooFewIdentities: a network needing more nodes than
// the document declares must say so, not compose a smaller one.
func TestDeclaredKeys_RefusesTooFewIdentities(t *testing.T) {
	_, err := store.DeclaredKeys{Path: t.TempDir(), Set: declaredSet(t, 2)}.Ensure(context.Background(), 4)
	if err == nil {
		t.Fatal("a set of 2 satisfied a network of 4")
	}
	if !strings.Contains(err.Error(), "needs 4") {
		t.Errorf("error %q does not say how many were needed", err)
	}
}

// TestDeclaredKeys_IsAKeySource keeps it on the boundary the composition and
// the engine both use. A source that did not satisfy it would be a second way
// to obtain keys, which is what this change exists to avoid.
func TestDeclaredKeys_IsAKeySource(t *testing.T) {
	var _ store.KeySource = store.DeclaredKeys{}
}

// TestDeclaredKeys_PinnedMismatchIsRefused is MON-002's contract. Reuse stays
// the rule — the ring on disk is what a genesis and the datadirs already refer
// to — but a key the operator PINNED to a different identity is a contradiction,
// not a preference to drop in silence. The refusal must leave the ring alone.
func TestDeclaredKeys_PinnedMismatchIsRefused(t *testing.T) {
	dir := t.TempDir()
	first := declaredSet(t, 2)
	if _, err := (store.DeclaredKeys{Path: dir, Set: first}).Ensure(context.Background(), 2); err != nil {
		t.Fatalf("first ensure: %v", err)
	}

	// node1 is pinned to a different identity than the one on disk (declaredSet
	// is deterministic, so the swap is what makes them differ).
	second := declaredSet(t, 2)
	second.Nodes[0].Nodekey = first.Nodes[1].Nodekey
	second.Nodes[0].Identity = first.Nodes[1].Identity
	_, err := store.DeclaredKeys{Path: dir, Set: second, Pinned: []int{1}}.Ensure(context.Background(), 2)
	if err == nil {
		t.Fatal("a pinned key that differs from the existing ring must be refused")
	}
	// Both addresses are public, and naming them is what tells the operator
	// which side to change.
	if !strings.Contains(err.Error(), first.Nodes[0].Address) {
		t.Fatalf("the refusal should name the identity on disk: %v", err)
	}

	// The ring on disk is untouched: the refusal changed nothing.
	after, lerr := store.PresetKeys{Path: dir}.Ensure(context.Background(), 2)
	if lerr != nil {
		t.Fatalf("reload: %v", lerr)
	}
	for i := range after.Nodes {
		if after.Nodes[i].Address != first.Nodes[i].Address {
			t.Fatalf("node%d changed to %s after a refusal", i+1, after.Nodes[i].Address)
		}
	}
}

// TestDeclaredKeys_PinnedMatchReuses: the same identity pinned again is not a
// conflict — it is the normal re-run, and it must keep working.
func TestDeclaredKeys_PinnedMatchReuses(t *testing.T) {
	dir := t.TempDir()
	first := declaredSet(t, 2)
	if _, err := (store.DeclaredKeys{Path: dir, Set: first}).Ensure(context.Background(), 2); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	got, err := store.DeclaredKeys{Path: dir, Set: first, Pinned: []int{1, 2}}.Ensure(context.Background(), 2)
	if err != nil {
		t.Fatalf("pinning the same keys must reuse them: %v", err)
	}
	if got.Nodes[0].Address != first.Nodes[0].Address {
		t.Fatalf("node1 = %s, want the existing %s", got.Nodes[0].Address, first.Nodes[0].Address)
	}
}

// TestDeclaredKeys_UnpinnedNodesStillReuse: a node the table left out carries
// fresh entropy on every call. Comparing it would fail every re-run, so only
// pinned indexes are checked.
func TestDeclaredKeys_UnpinnedNodesStillReuse(t *testing.T) {
	dir := t.TempDir()
	first := declaredSet(t, 2)
	if _, err := (store.DeclaredKeys{Path: dir, Set: first}).Ensure(context.Background(), 2); err != nil {
		t.Fatalf("first ensure: %v", err)
	}
	// node1 is pinned to what is on disk; node2 declares a different identity but
	// is NOT pinned, standing in for a node the table left out.
	second := declaredSet(t, 2)
	second.Nodes[1].Nodekey = first.Nodes[0].Nodekey
	second.Nodes[1].Identity = first.Nodes[0].Identity
	got, err := store.DeclaredKeys{Path: dir, Set: second, Pinned: []int{1}}.Ensure(context.Background(), 2)
	if err != nil {
		t.Fatalf("an unpinned node must not turn a re-run into a failure: %v", err)
	}
	if got.Nodes[1].Address != first.Nodes[1].Address {
		t.Fatalf("node2 = %s, want the existing %s", got.Nodes[1].Address, first.Nodes[1].Address)
	}
}
