package chainpreset

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// The golden profile pins the successor set's addresses, BLS public keys and
// the RLP extra-data that encodes them. Every one of those is derivable from
// the key set the handoff runs on, through the identity order the profile
// itself declares.
//
// So the block states a fact rather than making a choice, and stating a fact
// twice is a way to be wrong twice: edit the preset, or reorder plan_order, and
// the profile keeps asserting the old set. What follows is a chain whose
// genesis names validators that are not the nodes that came up, which surfaces
// as a network that syncs and never seals — a long way from the file that
// caused it.
//
// This is also what says the block is not a blocker for composing a handoff
// through the ordinary path: there is nothing in it the key set does not
// already know.

// TestPreset_TheValidatorSetIsWhatTheKeySetDerives.
func TestPreset_TheValidatorSetIsWhatTheKeySetDerives(t *testing.T) {
	prof, err := Load("../../presets/chain/wemix-upgrade.yaml")
	if err != nil {
		t.Fatalf("read the golden preset: %v", err)
	}
	preset, err := store.LoadPreset("../../presets/keys")
	if err != nil {
		t.Fatalf("read the shipped key set: %v", err)
	}
	order := prof.Identities.PlanOrder
	if len(order) < 2 {
		t.Fatalf("plan_order = %v, want the producer and at least one successor", order)
	}

	// Plan node 1 is the producer; the rest are the successors, which are the
	// to-chain's validators.
	var addrs, bls []string
	for _, presetNum := range order[1:] {
		e, ok := preset.Node(presetNum)
		if !ok {
			t.Fatalf("the key set has no node %d, which plan_order asks for", presetNum)
		}
		addrs = append(addrs, e.Address)
		if e.BLS == nil {
			t.Fatalf("preset node %d has no BLS key, and a wbft validator needs one", presetNum)
		}
		bls = append(bls, e.BLS.PublicKey)
	}

	assertSame(t, "validator addresses", addrs, prof.Validators.Addresses)
	assertSame(t, "bls public keys", bls, prof.Validators.BLSPublicKeys)

	// And the extra-data, which is the one a reader cannot check by eye: 300-odd
	// hex characters encoding the two lists above.
	xd, err := wbft.ExtraData(addrs, bls)
	if err != nil {
		t.Fatalf("derive extra-data: %v", err)
	}
	if !strings.EqualFold(xd, prof.Validators.ExtraData) {
		t.Errorf("extra_data does not encode the key set's validators\n derived %s\n profile %s", xd, prof.Validators.ExtraData)
	}
}

// assertSame compares two lists case-insensitively, since an address is hex and
// the profile and the key set need not agree on its casing.
func assertSame(t *testing.T, what string, derived, declared []string) {
	t.Helper()
	if len(derived) != len(declared) {
		t.Errorf("%s: derived %d, profile declares %d", what, len(derived), len(declared))
		return
	}
	for i := range derived {
		if !strings.EqualFold(derived[i], declared[i]) {
			t.Errorf("%s[%d]: derived %s, profile says %s", what, i, derived[i], declared[i])
		}
	}
}
