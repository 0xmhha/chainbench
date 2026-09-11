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
	if err := sameValidatorSet(genesisVals, keyVals, "genesis validators", "the running keys"); err != nil {
		return fmt.Errorf("chainsetup: genesis: existing genesis %s does not match the composed keys — %w", ref, err)
	}
	return nil
}

// sameValidatorSet reports whether two address lists are the same set,
// case-insensitively and order-independently. On a mismatch it names which
// addresses of each side the other is missing, labelled by aLabel/bLabel —
// addresses are public, so naming them is safe and is what an operator needs to
// fix the pairing. The two callers are the genesis-time check (genesis vs keys)
// and the runtime check (the validators the chain reports vs the composed keys).
func sameValidatorSet(a, b []string, aLabel, bLabel string) error {
	as, bs := addressSet(a), addressSet(b)
	var aOnly, bOnly []string
	for x := range as {
		if !bs[x] {
			aOnly = append(aOnly, x)
		}
	}
	for x := range bs {
		if !as[x] {
			bOnly = append(bOnly, x)
		}
	}
	if len(aOnly) == 0 && len(bOnly) == 0 {
		return nil
	}
	sort.Strings(aOnly)
	sort.Strings(bOnly)
	var parts []string
	if len(aOnly) > 0 {
		parts = append(parts, fmt.Sprintf("%s not in %s: %s", aLabel, bLabel, strings.Join(aOnly, ", ")))
	}
	if len(bOnly) > 0 {
		parts = append(parts, fmt.Sprintf("%s not in %s: %s", bLabel, aLabel, strings.Join(bOnly, ", ")))
	}
	return fmt.Errorf("%s (block signing would fail; pair the chain with the matching key set)", strings.Join(parts, "; "))
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
