package testhelper_test

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"github.com/0xmhha/chainbench/internal/testhelper"
)

// ring builds a key set holding two node identities and one dev account.
func ring(t *testing.T) *store.KeySet {
	t.Helper()
	r := store.NewKeySet(t.TempDir())
	for _, label := range []keyring.Label{"node1", "node2", "dev1"} {
		if _, err := r.Add(context.Background(), label, keyring.RandomSource{}, derive.AccountOnly); err != nil {
			t.Fatalf("Add %s: %v", label, err)
		}
	}
	return r
}

// TestResolveAccount_ALabelAnswersAddressAndSigner.
//
// A label has to answer both halves, because the two kinds of account are
// signed in different places: a node's account is unlocked in that node's
// keystore and the node signs for it, while an account only this harness holds
// must be signed here and submitted raw. A resolver that returned an address
// alone would leave every caller to work that out, and they would not all
// reach the same answer.
func TestResolveAccount_ALabelAnswersAddressAndSigner(t *testing.T) {
	r := ring(t)
	d := &interp.Deps{Keys: r}

	node1, _ := r.Get("node1")
	dev1, _ := r.Get("dev1")

	for _, tc := range []struct {
		ref       string
		wantAddr  string
		wantLocal bool
		why       string
	}{
		{"node1", node1.Address, false, "a node signs for its own account"},
		{"dev1", dev1.Address, true, "only this harness holds a dev key"},
		{"faucet", node1.Address, false, "the reserved name for the funded account"},
	} {
		got, err := testhelper.ResolveAccount(d, tc.ref)
		if err != nil {
			t.Fatalf("%s: %v", tc.ref, err)
		}
		if got.Address != tc.wantAddr {
			t.Errorf("%s -> %s, want %s", tc.ref, got.Address, tc.wantAddr)
		}
		if got.SignsLocally() != tc.wantLocal {
			t.Errorf("%s signs locally = %v, want %v (%s)", tc.ref, got.SignsLocally(), tc.wantLocal, tc.why)
		}
	}
}

// TestResolveAccount_AnAddressPassesThrough: a spec that already knows an
// address keeps working, and resolving it claims nothing about who holds the
// key.
func TestResolveAccount_AnAddressPassesThrough(t *testing.T) {
	got, err := testhelper.ResolveAccount(&interp.Deps{Keys: ring(t)}, "0x71562b71999873db5b286df957af199ec94617f7")
	if err != nil {
		t.Fatal(err)
	}
	if got.Address != "0x71562b71999873db5b286df957af199ec94617f7" || got.Label != "" || got.SignsLocally() {
		t.Fatalf("literal address resolved to %+v", got)
	}
}

// TestResolveAccount_AnUnknownNameIsAnError.
//
// The zero address accepts value and never complains, so a typo that fell
// through to it would send funds nowhere and pass. Bindings already refuse an
// unbound "$name" for the same reason; a label is no different.
func TestResolveAccount_AnUnknownNameIsAnError(t *testing.T) {
	_, err := testhelper.ResolveAccount(&interp.Deps{Keys: ring(t)}, "dev9")
	if err == nil {
		t.Fatal("an unknown label resolved instead of failing")
	}
	if !strings.Contains(err.Error(), "dev9") {
		t.Errorf("the error does not name the label: %v", err)
	}
	for _, want := range []string{"node1", "dev1", "faucet"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not say %q is available: %v", want, err)
		}
	}
}

// TestResolveAccount_IsDeterministic: the same label against the same key set
// must give the same address every time, or every artifact derived from it
// disagrees with the network it describes.
func TestResolveAccount_IsDeterministic(t *testing.T) {
	d := &interp.Deps{Keys: ring(t)}
	first, err := testhelper.ResolveAccount(d, "dev1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := testhelper.ResolveAccount(d, "dev1")
	if err != nil {
		t.Fatal(err)
	}
	if first.Address != second.Address {
		t.Fatalf("dev1 resolved to %s then %s", first.Address, second.Address)
	}
}
