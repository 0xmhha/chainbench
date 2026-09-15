package testhelper

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// depsWithRing builds a key set holding two node identities and one dev account.
func depsWithRing(t *testing.T) (*interp.Deps, *store.KeySet) {
	t.Helper()
	r := store.NewKeySet(t.TempDir())
	for _, label := range []keyring.Label{"node1", "node2", "dev1"} {
		if _, err := r.Add(context.Background(), label, keyring.RandomSource{}, derive.AccountOnly); err != nil {
			t.Fatalf("Add %s: %v", label, err)
		}
	}
	return &interp.Deps{Keys: r}, r
}

// TestResolveAddressArgs_ResolvesLabelsInAValueList is why "of" was added.
//
// derive's abiCall packs its arguments into 32-byte words, and one of them is
// usually an account. Until the list was resolved, that was the one place a
// spec still had to paste hex to say "this account" — the labels stopped at the
// argument names, so half a case read node1 and the other half read the address
// node1 happens to have.
func TestResolveAddressArgs_ResolvesLabelsInAValueList(t *testing.T) {
	d, r := depsWithRing(t)
	node2, _ := r.Get("node2")

	spec := map[string]any{
		"source": "derive", "op": "abiCall", "selector": "0xb03d36cd",
		"of": []any{"node2"},
	}
	out, err := resolveAddressArgs(d, spec)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := out["of"].([]any)
	if len(got) != 1 || got[0] != node2.Address {
		t.Fatalf("of = %v, want [%s]", got, node2.Address)
	}
	// The input is read again on every target node, so it must not be rewritten
	// in place.
	if in, _ := spec["of"].([]any); in[0] != "node2" {
		t.Errorf("the caller's spec was modified: %v", in)
	}
}

// TestResolveAddressArgs_LeavesAListItCannotResolve: "of" carries numbers, hex
// blobs and already-substituted bindings as well as accounts. An element that
// is none of those is reported by whatever consumes the list, with the message
// that knows what it expected — resolving here must not turn that into a
// different error, or into silence.
func TestResolveAddressArgs_LeavesAListItCannotResolve(t *testing.T) {
	d, _ := depsWithRing(t)

	for _, tc := range []struct {
		name string
		of   []any
	}{
		{"numbers", []any{"1", "2"}},
		{"a hex blob", []any{"0xdeadbeef"}},
		{"a name in no key set", []any{"nosuchlabel"}},
		{"mixed with no account", []any{"0x1", float64(2)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			spec := map[string]any{"of": tc.of}
			out, err := resolveAddressArgs(d, spec)
			if err != nil {
				t.Fatalf("an unresolvable element must not be an error here: %v", err)
			}
			got, _ := out["of"].([]any)
			for i := range tc.of {
				if got[i] != tc.of[i] {
					t.Errorf("of[%d] = %v, want it left as %v", i, got[i], tc.of[i])
				}
			}
		})
	}
}

// TestResolveAddressArgs_AnUnknownLabelInAnAddressArgumentStillFails guards the
// asymmetry the two lists carry on purpose. An address argument MUST be an
// account, so a typo there has to fail the step rather than send value nowhere;
// a value list may hold anything, so it may not.
func TestResolveAddressArgs_AnUnknownLabelInAnAddressArgumentStillFails(t *testing.T) {
	d, _ := depsWithRing(t)

	if _, err := resolveAddressArgs(d, map[string]any{"from": "nosuchlabel"}); err == nil {
		t.Fatal("an unknown account in \"from\" must fail")
	}
	if _, err := resolveAddressArgs(d, map[string]any{"of": []any{"nosuchlabel"}}); err != nil {
		t.Fatalf("an unknown name in \"of\" must be left alone: %v", err)
	}
}
