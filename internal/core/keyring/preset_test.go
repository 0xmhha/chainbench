package keyring_test

import (
	"bytes"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// bareRing generates a ring that declares nothing about a network: identities
// and no validator set.
func bareRing(t *testing.T, nodes int) keyring.Preset {
	t.Helper()
	none := 0
	set, err := store.Generate(store.GenerateOpts{
		Nodes: nodes, Validators: &none,
		Out: t.TempDir(), Password: "1", Balance: "0x1",
		Derive: derive.WithBLS,
		Rand:   bytes.NewReader(bytes.Repeat([]byte{0x5a}, nodes*derive.PrivateKeyLen)),
	}, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return set
}

// TestNetworkFor_BareRingIsIdentitiesOnly pins the bare-ring contract: a ring can hold
// identities and say nothing about a network, so a preset is a choice rather
// than a premise.
func TestNetworkFor_BareRingIsIdentitiesOnly(t *testing.T) {
	set := bareRing(t, 4)
	if len(set.Nodes) != 4 {
		t.Fatalf("got %d identities", len(set.Nodes))
	}
	if len(set.Network.Validators) != 0 {
		t.Errorf("a bare ring declared a validator set: %v", set.Network.Validators)
	}

	// The network decides, and gets a usable answer.
	cases := []struct {
		name string
		ask  int
		want int
	}{
		{name: "fewer than the ring holds", ask: 2, want: 2},
		{name: "all of them", ask: 0, want: 4},
		{name: "more than the ring holds", ask: 9, want: 4},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			net := set.NetworkFor(tc.ask)
			if len(net.Validators) != tc.want {
				t.Fatalf("got %d validators, want %d", len(net.Validators), tc.want)
			}
			if len(net.BLSKeys) != tc.want {
				t.Errorf("got %d BLS keys, want %d", len(net.BLSKeys), tc.want)
			}
			// They are the ring's first identities, in ring order.
			for i, addr := range net.Validators {
				if addr != set.Nodes[i].Address {
					t.Errorf("validator %d is %s, want the ring's node%d", i+1, addr, i+1)
				}
			}
			// A council with no members cannot pass anything, so one is seated.
			if len(net.Members) != tc.want {
				t.Errorf("got %d council members, want %d", len(net.Members), tc.want)
			}
		})
	}
}

// TestNetworkFor_DeclaredSetWins keeps an existing preset's answer: a file that
// says who validates is not second-guessed.
func TestNetworkFor_DeclaredSetWins(t *testing.T) {
	p, err := store.LoadPreset(filepath.Join("..", "..", "..", "keys", "preset"))
	if err != nil {
		t.Fatalf("LoadPreset: %v", err)
	}
	declared := p.Network.Validators
	if len(declared) == 0 {
		t.Skip("the shipped preset declares no validators")
	}
	// The shipped ring holds more identities than it declares validators, so a
	// derived answer would differ from the declared one.
	if len(p.Nodes) <= len(declared) {
		t.Skip("the shipped preset declares every identity as a validator")
	}
	net := p.NetworkFor(0)
	if len(net.Validators) != len(declared) {
		t.Errorf("got %d validators, want the declared %d", len(net.Validators), len(declared))
	}
}

// TestLoadPreset_RejectsAFileThatSaysNothing keeps an empty file from loading as
// an empty ring, which would produce a genesis with no producers.
func TestLoadPreset_RejectsAFileThatSaysNothing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, store.PresetFile), []byte(`{"password":"1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadPreset(dir); err == nil {
		t.Fatal("loaded a ring that holds no identities and declares no validators")
	}
}

// TestValidatorCount_UnsetMeansOppositeThingsPerVerb pins the one decision that
// has been got wrong twice: what an unset validator count means.
//
// It means all of them when a ring is created and none when one is extended,
// and the two live here rather than in a caller — a caller that resolves it has
// to know which verb it is about to call, and will eventually resolve it for
// the wrong one. The field is a pointer so that "unset" is not the same value
// as "none".
func TestValidatorCount_UnsetMeansOppositeThingsPerVerb(t *testing.T) {
	none, two := 0, 2

	cases := []struct {
		name       string
		validators *int
		// wantNew is the validator count after creating a 3-identity ring.
		wantNew int
		// wantAdded is how many validators adding 2 identities appends.
		wantAdded int
	}{
		{name: "unset", validators: nil, wantNew: 3, wantAdded: 0},
		{name: "explicitly none", validators: &none, wantNew: 0, wantAdded: 0},
		{name: "explicitly two", validators: &two, wantNew: 2, wantAdded: 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			created, err := store.Generate(store.GenerateOpts{
				Nodes: 3, Validators: tc.validators, Out: dir,
				Password: "1", Balance: "0x1", Derive: derive.WithBLS,
				Rand: bytes.NewReader(bytes.Repeat([]byte{0x33}, 3*derive.PrivateKeyLen)),
			}, nil)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if got := len(created.Network.Validators); got != tc.wantNew {
				t.Errorf("new: %d validators, want %d", got, tc.wantNew)
			}

			extended, err := store.Extend(store.GenerateOpts{
				Nodes: 2, Validators: tc.validators, Out: dir,
				Password: "1", Balance: "0x1", Derive: derive.WithBLS,
				Rand: bytes.NewReader(bytes.Repeat([]byte{0x44}, 2*derive.PrivateKeyLen)),
			}, nil)
			if err != nil {
				t.Fatalf("Extend: %v", err)
			}
			if got := len(extended.Nodes); got != 5 {
				t.Fatalf("extend: %d identities, want 5", got)
			}
			added := len(extended.Network.Validators) - len(created.Network.Validators)
			if added != tc.wantAdded {
				t.Errorf("extend promoted %d identities, want %d", added, tc.wantAdded)
			}
		})
	}
}

// TestExtend_RejectsMoreValidatorsThanIdentities keeps a count that cannot be
// satisfied from silently promoting fewer.
func TestExtend_RejectsMoreValidatorsThanIdentities(t *testing.T) {
	dir := t.TempDir()
	if _, err := store.Generate(store.GenerateOpts{
		Nodes: 2, Out: dir, Password: "1", Balance: "0x1",
		Rand: bytes.NewReader(bytes.Repeat([]byte{0x55}, 2*derive.PrivateKeyLen)),
	}, nil); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	tooMany := 3
	_, err := store.Extend(store.GenerateOpts{
		Nodes: 1, Validators: &tooMany, Out: dir, Password: "1", Balance: "0x1",
	}, nil)
	if err == nil {
		t.Fatal("Extend accepted 3 new validators from 1 new identity")
	}
}
