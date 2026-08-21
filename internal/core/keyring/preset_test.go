package keyring_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
)

// bareRing generates a ring that declares nothing about a network: identities
// and no validator set.
func bareRing(t *testing.T, nodes int) keyring.Preset {
	t.Helper()
	set, err := keyring.Generate(keyring.GenerateOpts{
		Nodes: nodes, Validators: keyring.NoValidators,
		Out: t.TempDir(), Password: "1", Balance: "0x1",
		Derive: keyring.WithBLS,
		Rand:   bytes.NewReader(bytes.Repeat([]byte{0x5a}, nodes*keyring.PrivateKeyLen)),
	}, nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	return set
}

// TestNetworkFor_BareRingIsIdentitiesOnly is the K5 gate: a ring can hold
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
	p, err := keyring.LoadPreset(filepath.Join("..", "..", "..", "keys", "preset"))
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
	if err := os.WriteFile(filepath.Join(dir, keyring.PresetFile), []byte(`{"password":"1"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := keyring.LoadPreset(dir); err == nil {
		t.Fatal("loaded a ring that holds no identities and declares no validators")
	}
}
