package chainsetup

import (
	"fmt"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// verifyExistingGenesisKeys checks that the validators an existing genesis
// encodes are exactly the validator addresses the composed key set provides.
//
// A generated genesis is built from the key set, so the two cannot disagree.
// An existing genesis is used verbatim, so it can — and a validator whose
// address is not one of the running node keys cannot sign, which passes genesis
// validation and then stalls consensus with the cause far from the symptom.
// This turns that into a refusal at compose time.
//
// It runs for a family that carries its validator set in the genesis (wbft, in
// the extra-data). A family that keeps the set elsewhere (poa, in a governance
// config) does not implement the reader, and the check is skipped for it.
func (w *Workspace) verifyExistingGenesisKeys(p registry.ChainPlugin, genesisJSON []byte, ref string) error {
	reader, ok := p.Family().(registry.GenesisValidatorReader)
	if !ok {
		return nil
	}
	genesisVals, err := reader.GenesisValidators(genesisJSON)
	if err != nil {
		return fmt.Errorf("chainsetup: genesis: existing genesis %s: %w", ref, err)
	}
	preset, err := store.LoadPreset(w.state.KeysDir)
	if err != nil {
		return fmt.Errorf("chainsetup: genesis: load keys to verify against the existing genesis: %w", err)
	}
	keyVals := preset.NetworkFor(w.state.Validators).Validators
	if err := sameValidatorSet(genesisVals, keyVals); err != nil {
		return fmt.Errorf("chainsetup: genesis: existing genesis %s does not match the composed keys — %w", ref, err)
	}
	return nil
}

// sameValidatorSet reports whether two address lists are the same set,
// case-insensitively and order-independently. On a mismatch it names which
// addresses the genesis expects that no key provides, and which keys are not in
// the genesis — addresses are public, so naming them is safe and is what an
// operator needs to fix the pairing.
func sameValidatorSet(genesis, keys []string) error {
	g := addressSet(genesis)
	k := addressSet(keys)
	var missingKey, missingGenesis []string
	for a := range g {
		if !k[a] {
			missingKey = append(missingKey, a)
		}
	}
	for a := range k {
		if !g[a] {
			missingGenesis = append(missingGenesis, a)
		}
	}
	if len(missingKey) == 0 && len(missingGenesis) == 0 {
		return nil
	}
	sort.Strings(missingKey)
	sort.Strings(missingGenesis)
	var parts []string
	if len(missingKey) > 0 {
		parts = append(parts, fmt.Sprintf("genesis validators with no running key: %s", strings.Join(missingKey, ", ")))
	}
	if len(missingGenesis) > 0 {
		parts = append(parts, fmt.Sprintf("running keys not in the genesis validator set: %s", strings.Join(missingGenesis, ", ")))
	}
	return fmt.Errorf("%s (block signing would fail; pair the genesis with the matching key set)", strings.Join(parts, "; "))
}

// addressSet lowercases each address into a set for order-independent
// comparison.
func addressSet(addrs []string) map[string]bool {
	m := make(map[string]bool, len(addrs))
	for _, a := range addrs {
		m[strings.ToLower(a)] = true
	}
	return m
}
