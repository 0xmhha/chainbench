package testhelper

import (
	"context"
	"strings"
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

// TestResolveAddress_NamesAContractTheChainDeclares.
//
// The address does not identify a contract: 0x…1001 is govValidator on
// stablenet and govStaking on wbft. A spec that writes the address calls a
// different contract the moment it runs anywhere else, and nothing says so.
func TestResolveAddress_NamesAContractTheChainDeclares(t *testing.T) {
	d, r := depsWithRing(t)
	d.Contracts = map[string]string{"govMinter": "0x0000000000000000000000000000000000001003"}
	node1, _ := r.Get("node1")

	for _, tc := range []struct{ ref, want string }{
		{"govMinter", "0x0000000000000000000000000000000000001003"},
		{"node1", node1.Address},
		{"0x00000000000000000000000000000000c0ffee01", "0x00000000000000000000000000000000c0ffee01"},
	} {
		got, err := ResolveAddress(d, tc.ref)
		if err != nil {
			t.Errorf("%s: %v", tc.ref, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%s resolved to %s, want %s", tc.ref, got, tc.want)
		}
	}
}

// TestResolveAddress_SaysBothPlacesItLooked: a name that is neither is a typo,
// and the reader has to learn which of the two vocabularies they missed.
func TestResolveAddress_SaysBothPlacesItLooked(t *testing.T) {
	d, _ := depsWithRing(t)
	d.Contracts = map[string]string{"govMinter": "0x0000000000000000000000000000000000001003"}

	_, err := ResolveAddress(d, "govMintr")
	if err == nil {
		t.Fatal("an unknown name must be refused")
	}
	for _, want := range []string{"node1", "govMinter"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal must list %s: %v", want, err)
		}
	}
}

// TestResolveAddress_AChainWithNoContractsSaysSo: wemix declares none because it
// deploys them at run time. "this chain has no govMinter" and "this build does
// not know the chain" lead to different next steps, so the message must not read
// as a missing name.
func TestResolveAddress_AChainWithNoContractsSaysSo(t *testing.T) {
	d, _ := depsWithRing(t)

	_, err := ResolveAddress(d, "govMinter")
	if err == nil {
		t.Fatal("a chain with no contract table cannot resolve a contract name")
	}
	if !strings.Contains(err.Error(), "no contracts") {
		t.Errorf("message must say the chain declares none: %v", err)
	}
}

// TestResolveAddressArgs_ASignerIsNeverAContract.
//
// A contract has no key. Letting its name stand in a signing position would
// hand a node an address it cannot sign for, and the failure would come back
// from the node as "unknown account" — about the address, not about the name
// that produced it.
func TestResolveAddressArgs_ASignerIsNeverAContract(t *testing.T) {
	d, _ := depsWithRing(t)
	d.Contracts = map[string]string{"govMinter": "0x0000000000000000000000000000000000001003"}

	if _, err := resolveAddressArgs(d, map[string]any{"from": "govMinter"}); err == nil {
		t.Error("a contract name in \"from\" must be refused")
	}
	// The same name in an address position resolves.
	out, err := resolveAddressArgs(d, map[string]any{"to": "govMinter"})
	if err != nil {
		t.Fatalf("\"to\" must accept a contract: %v", err)
	}
	if out["to"] != "0x0000000000000000000000000000000000001003" {
		t.Errorf("to = %v", out["to"])
	}
}

// TestResolveAccount_TellsAContractFromATypo.
//
// A contract name reaching an account position is a wiring mistake: some
// argument resolves through the account path when it should take an address.
// That happened — sendTx resolved "to" with ResolveAccount, so a case naming a
// contract there failed with "unknown account govValidator", which reads as a
// missing key and sends the reader looking in the wrong place.
func TestResolveAccount_TellsAContractFromATypo(t *testing.T) {
	d, _ := depsWithRing(t)
	d.Contracts = map[string]string{"govValidator": "0x0000000000000000000000000000000000001001"}

	_, err := ResolveAccount(d, "govValidator")
	if err == nil {
		t.Fatal("a contract is not an account")
	}
	for _, want := range []string{"contracts", "no key"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the message must say it is a contract without a key: %v", err)
		}
	}
	// A name that is neither still reads as a typo.
	_, err = ResolveAccount(d, "nosuchthing")
	if err == nil || !strings.Contains(err.Error(), "unknown account") {
		t.Errorf("a plain typo must still say unknown account: %v", err)
	}
}
