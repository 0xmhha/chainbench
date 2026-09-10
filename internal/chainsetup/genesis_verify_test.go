package chainsetup

import (
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins
	"github.com/0xmhha/chainbench/internal/chains/external"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

const presetDir = "../../keys/preset"

// TestVerifyExistingGenesisKeys_MatchesAndMismatches drives the wbft-family
// check against the committed preset: a genesis whose extra-data encodes the
// preset's own validators passes, and a key set that is missing one of them
// fails.
func TestVerifyExistingGenesisKeys_MatchesAndMismatches(t *testing.T) {
	p, err := external.ResolveChain("stablenet", "", "")
	if err != nil {
		t.Fatalf("resolve stablenet: %v", err)
	}
	preset, err := store.LoadPreset(presetDir)
	if err != nil {
		t.Fatalf("load preset: %v", err)
	}
	full := len(preset.Network.Validators)
	if full < 2 || preset.Network.ExtraData == "" {
		t.Fatalf("preset needs >=2 validators and a recorded extraData (got %d, extraData=%q)", full, preset.Network.ExtraData)
	}
	genesis := []byte(`{"extraData":"` + preset.Network.ExtraData + `"}`)

	// The whole key set matches the genesis's validators.
	w := &Workspace{state: State{KeysDir: presetDir, Validators: full}}
	if err := w.verifyExistingGenesisKeys(p, genesis, "preset.json"); err != nil {
		t.Fatalf("matching key set should pass: %v", err)
	}

	// One fewer validator in the key set than the genesis names -> a genesis
	// validator with no running key -> refused.
	wShort := &Workspace{state: State{KeysDir: presetDir, Validators: full - 1}}
	if err := wShort.verifyExistingGenesisKeys(p, genesis, "preset.json"); err == nil {
		t.Fatal("a key set missing a genesis validator must be refused")
	}
}

func TestSameValidatorSet(t *testing.T) {
	// Same set, different order and case -> equal.
	if err := sameValidatorSet([]string{"0xAA", "0xbb"}, []string{"0xbb", "0xaa"}); err != nil {
		t.Fatalf("same set should match: %v", err)
	}
	// A genesis validator with no key.
	if err := sameValidatorSet([]string{"0xaa", "0xbb"}, []string{"0xaa"}); err == nil {
		t.Fatal("missing key must error")
	}
	// A key not in the genesis.
	if err := sameValidatorSet([]string{"0xaa"}, []string{"0xaa", "0xcc"}); err == nil {
		t.Fatal("extra key must error")
	}
}
