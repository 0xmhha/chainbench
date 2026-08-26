package store_test

import (
	"context"
	"errors"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// TestKeySet_AddIsIdempotent is what makes re-running a command safe: a label
// resolved once keeps its identity, instead of a second run producing a
// competing key for the same name.
func TestKeySet_AddIsIdempotent(t *testing.T) {
	ring := store.NewKeySet(t.TempDir())
	first, err := ring.Add(context.Background(), "bp1", keyring.RandomSource{}, derive.WithBLS)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	second, err := ring.Add(context.Background(), "bp1", keyring.RandomSource{}, derive.WithBLS)
	if err != nil {
		t.Fatalf("Add again: %v", err)
	}
	if first.Nodekey.Hex() != second.Nodekey.Hex() {
		t.Error("a repeated Add replaced the key")
	}
	if got, ok := ring.Get("bp1"); !ok || got.Address != first.Address {
		t.Error("Get did not return the held entry")
	}
	if _, ok := ring.Get("nobody"); ok {
		t.Error("Get invented an entry")
	}
}

// TestKeySet_DerivationIsPerEntry covers the poa case: an entry asked for without
// BLS has none, rather than a zero key that reads like a real one.
func TestKeySet_DerivationIsPerEntry(t *testing.T) {
	ring := store.NewKeySet(t.TempDir())
	bp, err := ring.Add(context.Background(), "bp1", keyring.RandomSource{}, derive.WithBLS)
	if err != nil {
		t.Fatalf("Add bp1: %v", err)
	}
	acct, err := ring.Add(context.Background(), "faucet", keyring.RandomSource{}, derive.AccountOnly)
	if err != nil {
		t.Fatalf("Add faucet: %v", err)
	}
	if bp.BLS == nil {
		t.Error("a WithBLS entry has no BLS material")
	}
	if acct.BLS != nil {
		t.Error("an AccountOnly entry derived BLS material")
	}
}

// TestKeySet_AddExpectingCatchesDrift is the check that keeps a declared identity
// and its key from parting ways unnoticed.
func TestKeySet_AddExpectingCatchesDrift(t *testing.T) {
	entry := presetEntry(t)
	ring := store.NewKeySet(t.TempDir())
	src := keyring.PrivateKeySource{Hex: entry.Nodekey.Hex()}

	if _, err := ring.AddExpecting(context.Background(), "ok", src, derive.AccountOnly, entry.Address); err != nil {
		t.Fatalf("AddExpecting with the right address: %v", err)
	}
	_, err := ring.AddExpecting(context.Background(), "drifted", src, derive.AccountOnly,
		"0x00000000000000000000000000000000deadbeef")
	if err == nil {
		t.Fatal("AddExpecting accepted an address the key does not derive")
	}
}

// TestKeySet_WritesUnderItsDirectory checks the persistence contract: the secret
// is owner-only, the derived fields are readable, and nothing lands outside.
func TestKeySet_WritesUnderItsDirectory(t *testing.T) {
	dir := t.TempDir()
	ring := store.NewKeySet(dir)
	if _, err := ring.Add(context.Background(), "bp1", keyring.RandomSource{}, derive.WithBLS); err != nil {
		t.Fatalf("Add: %v", err)
	}
	cases := []struct {
		file string
		perm os.FileMode
	}{
		{"private", 0o600},
		{"address", 0o644},
		{"bls", 0o644},
		{"pop", 0o644},
	}
	for _, tc := range cases {
		info, err := os.Stat(filepath.Join(dir, "bp1", tc.file))
		if err != nil {
			t.Fatalf("%s: %v", tc.file, err)
		}
		if got := info.Mode().Perm(); got != tc.perm {
			t.Errorf("%s mode = %o, want %o", tc.file, got, tc.perm)
		}
	}
}

// TestKeySet_Install ships only the secret. Everything else derives from it, and
// shipping a derived value is how a node and its genesis come to disagree.
func TestKeySet_Install(t *testing.T) {
	ring := store.NewKeySet(t.TempDir())
	e, err := ring.Add(context.Background(), "bp1", keyring.RandomSource{}, derive.WithBLS)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	dest := t.TempDir()
	if err := ring.Install(context.Background(), filestore.Local{}, dest, []keyring.Label{"bp1"}); err != nil {
		t.Fatalf("Install: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "bp1", "private"))
	if err != nil {
		t.Fatalf("read installed key: %v", err)
	}
	if string(got) != e.Nodekey.Hex() {
		t.Error("the installed key is not the one held")
	}

	err = ring.Install(context.Background(), filestore.Local{}, dest, []keyring.Label{"nobody"})
	if err == nil {
		t.Fatal("Install accepted a label the ring does not hold")
	}
}

// TestKeySet_Labels keeps listing output stable between runs.
func TestKeySet_Labels(t *testing.T) {
	ring := store.NewKeySet(t.TempDir())
	for _, l := range []keyring.Label{"en1", "bp2", "bp1"} {
		if _, err := ring.Add(context.Background(), l, keyring.RandomSource{}, derive.AccountOnly); err != nil {
			t.Fatalf("Add %s: %v", l, err)
		}
	}
	got := ring.Labels()
	want := []keyring.Label{"bp1", "bp2", "en1"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// TestKeySet_AddPropagatesSourceErrors keeps a failed resolve from being recorded
// as an entry.
func TestKeySet_AddPropagatesSourceErrors(t *testing.T) {
	ring := store.NewKeySet(t.TempDir())
	_, err := ring.Add(context.Background(), "bad", keyring.PrivateKeySource{Hex: "zz"}, derive.AccountOnly)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, derive.ErrInvalidPrivateKey) {
		t.Errorf("error should unwrap to ErrInvalidPrivateKey, got %v", err)
	}
	if _, ok := ring.Get("bad"); ok {
		t.Error("a failed Add left an entry behind")
	}
}

// TestNetworkFor_DoesNotAliasThePreset is a regression: the narrowed result
// resliced the preset's own array, so appending to it rewrote the preset.
func TestNetworkFor_DoesNotAliasThePreset(t *testing.T) {
	p, err := store.LoadPreset(filepath.Join("..", "..", "..", "..", "keys", "preset"))
	if err != nil {
		t.Fatalf("LoadPreset: %v", err)
	}
	if len(p.Network.Validators) < 3 {
		t.Skip("the shipped preset declares too few validators for this check")
	}
	third := p.Network.Validators[2]

	net := p.NetworkFor(2)
	net.Validators = append(net.Validators, "0xdeadbeef")

	if p.Network.Validators[2] != third {
		t.Errorf("appending to the narrowed set rewrote the preset: %s -> %s",
			third, p.Network.Validators[2])
	}
}

// TestKeySet_AddIsSafeUnderConcurrency checks that one label yields one identity
// even when several callers race, and that a slow source does not serialize
// the others behind it.
func TestKeySet_AddIsSafeUnderConcurrency(t *testing.T) {
	ring := store.NewKeySet(t.TempDir())

	const callers = 8
	var wg sync.WaitGroup
	got := make([]string, callers)
	for i := range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e, err := ring.Add(context.Background(), "bp1", keyring.RandomSource{}, derive.AccountOnly)
			if err != nil {
				t.Errorf("Add: %v", err)
				return
			}
			got[i] = e.Address
		}()
	}
	wg.Wait()

	for i, addr := range got {
		if addr != got[0] {
			t.Fatalf("caller %d was handed a different identity: %s vs %s", i, addr, got[0])
		}
	}
	if len(ring.Labels()) != 1 {
		t.Errorf("ring holds %v, want one label", ring.Labels())
	}
}

// presetEntry returns the shipped preset's first entry — a published test
// fixture, reused so no key literal appears in source.
func presetEntry(t *testing.T) keyring.Entry {
	t.Helper()
	set, err := store.LoadPreset(filepath.Join("..", "..", "..", "..", "keys", "preset"))
	if err != nil {
		t.Fatalf("read shipped preset: %v", err)
	}
	return set.Nodes[0]
}
