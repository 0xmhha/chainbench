package store_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/preset"
)

// shippedPreset is the key set this repository ships.
var shippedPreset = filepath.Join("..", "..", "..", "..", "presets", "keys")

// TestLoadPresetWithAccounts_ANodeSealsWithWhatItsKeystoreHolds.
//
// A node's identity is its devp2p key; the account it seals and stakes with is
// a keystore. They coincide in a ring written consistently and they do not have
// to — the shipped preset's node5 holds another account on purpose, because that
// is a handoff's producer.
//
// Both halves of a launch read this. The genesis funds and stakes the account,
// and the node unlocks it; answering differently gave a producer that sealed
// nothing ("no key for given address or file") and then, once it could unlock,
// one that sealed with no balance ("insufficient funds for gas * price + value").
// Both were measured before this existed.
func TestLoadPresetWithAccounts_ANodeSealsWithWhatItsKeystoreHolds(t *testing.T) {
	set, err := preset.LoadKeyPresetWithAccounts(shippedPreset)
	if err != nil {
		t.Fatalf("read the shipped key set: %v", err)
	}

	five, ok := set.Node(5)
	if !ok {
		t.Fatal("the key set has no node5")
	}
	if five.Account == "" {
		t.Fatal("node5's keystore holds another account and the ring did not say so")
	}
	if strings.EqualFold(five.Account, five.Address) {
		t.Errorf("Account is set to the nodekey's own address (%s); it records only a difference", five.Address)
	}
	if got := five.SealingAccount(); !strings.EqualFold(got, five.Account) {
		t.Errorf("node5 seals with %s, want its keystore's %s", got, five.Account)
	}

	// And a consistent entry records no difference and answers with its own
	// address, which is every network composed from a ring written in one go.
	one, ok := set.Node(1)
	if !ok {
		t.Fatal("the key set has no node1")
	}
	if one.Account != "" {
		t.Errorf("node1 recorded an account difference it does not have: %s", one.Account)
	}
	if got := one.SealingAccount(); !strings.EqualFold(got, one.Address) {
		t.Errorf("node1 seals with %s, want its own %s", got, one.Address)
	}
}

// TestLoadPreset_SaysNothingAboutAccounts keeps the cheap read cheap: the index
// is one file and answers nearly every question a ring is asked. Only the two
// callers that have to agree about the sealing account pay for the keystores.
func TestLoadPreset_SaysNothingAboutAccounts(t *testing.T) {
	set, err := preset.LoadKeyPreset(shippedPreset)
	if err != nil {
		t.Fatalf("read the shipped key set: %v", err)
	}
	for _, e := range set.Nodes {
		if e.Account != "" {
			t.Errorf("node%d carried an account from the plain read", e.Index)
		}
	}
}
