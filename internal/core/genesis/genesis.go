// Package genesis builds a chain's genesis.json by delegating to the chain's
// consensus family (ConsensusFamily.BuildGenesis): the wbft family substitutes
// the template with the validator set and chain id; the poa family renders a
// base genesis whose membership is set at bootstrap. Dispatch is virtual through
// the registry contract, so this package imports no concrete family — the
// core/consensus boundary stays compiler-enforced (docs §3, §8; #13).
package genesis

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/registry"
)

// ErrNoStaticGenesis is returned for chains whose genesis is not built from a
// static template (the poa/registry family bootstraps its genesis at deploy
// time).
var ErrNoStaticGenesis = errors.New("genesis: chain has no static template")

// Inputs are the per-network genesis values the setup phase supplies (from the
// preset key metadata): the validator set and the RLP extra-data that encodes
// it.
type Inputs struct {
	Validators []string        // validator addresses (0x-hex) — wbft family
	BLSKeys    []string        // BLS public keys (0x-hex), aligned with Validators
	ExtraData  string          // validator extra-data (0x-hex) — wbft family
	Members    []string        // governance council addresses (0x-hex) — anzeon system contracts
	Alloc      json.RawMessage // raw genesis pre-funded accounts (address -> account) — wbft family
	Coinbase   string          // block coinbase (0x-hex) — poa family; default zero
	// ChainID, when non-zero, overrides the manifest chain id — the seam that
	// lets a test run a chain under a custom id without editing the manifest.
	ChainID int64
}

// Build produces the genesis.json bytes for a chain plugin by delegating to its
// consensus family. The chain id comes from the manifest; validators/BLS/extra
// data/members/alloc/coinbase come from in. Core does not branch on the family
// — the family owns the substitution.
func Build(p registry.ChainPlugin, in Inputs) ([]byte, error) {
	m := p.Manifest()
	tmpl := p.GenesisTemplate()
	if len(tmpl) == 0 {
		return nil, fmt.Errorf("genesis: chain %q (%s) has no template: %w", m.ID, m.ConsensusFamily, ErrNoStaticGenesis)
	}
	chainID := m.ChainID
	if in.ChainID != 0 {
		chainID = in.ChainID
	}
	return p.Family().BuildGenesis(tmpl, registry.GenesisParams{
		ChainID:    chainID,
		Validators: in.Validators,
		BLSKeys:    in.BLSKeys,
		ExtraData:  in.ExtraData,
		Members:    in.Members,
		Alloc:      in.Alloc,
		Coinbase:   in.Coinbase,
	})
}

// NetworkOptions are the optional post-Build genesis transforms a network launch
// applies: config overrides (the delayed-fork seam, e.g. {"bohoBlock":"10"}) and
// a deep-merge overlay (extra alloc or system-contract fragments). A zero value
// applies neither.
type NetworkOptions struct {
	// ConfigOverrides sets keys in the genesis `config` object. Empty applies
	// none.
	ConfigOverrides map[string]string
	// Overlay is a JSON fragment deep-merged into the built genesis. Empty
	// applies none.
	Overlay []byte
}

// BuildNetwork builds a chain's genesis and applies a launch's optional config
// overrides and overlay, revalidating fork ordering after each transform so a
// bad delayed-fork or overlay fails at build time rather than at node boot. Fork
// ordering is checked only after a transform — a plain build is passed through
// untouched. It is the single genesis composition the setup path and the engine
// launch path share.
func BuildNetwork(p registry.ChainPlugin, in Inputs, opts NetworkOptions) ([]byte, error) {
	gen, err := Build(p, in)
	if err != nil {
		return nil, err
	}
	return Customize(gen, opts)
}

// Customize applies a caller's genesis changes to an already-built genesis:
// bare config overrides first, then the overlay fragment, re-validating fork
// ordering after each so a bad request fails while composing rather than when a
// node refuses to boot.
//
// It is separate from Build because a genesis is not always built the same way.
// A family whose genesis its own binary writes produces bytes that Build never
// touches, and those bytes still have to accept the same overrides and the same
// overlay. Keeping the customization in Build meant only the families that went
// through Build could be customized — and a second copy of these very steps
// grew in the step surface to cover the other one.
func Customize(gen []byte, opts NetworkOptions) ([]byte, error) {
	var err error
	if len(opts.ConfigOverrides) > 0 {
		gen, err = ApplyConfigOverrides(gen, opts.ConfigOverrides)
		if err != nil {
			return nil, err
		}
		if err := ValidateForks(gen); err != nil {
			return nil, fmt.Errorf("genesis: overrides: %w", err)
		}
	}
	if len(opts.Overlay) > 0 {
		gen, err = MergeOverride(gen, opts.Overlay)
		if err != nil {
			return nil, err
		}
		if err := ValidateForks(gen); err != nil {
			return nil, fmt.Errorf("genesis: overlay: %w", err)
		}
	}
	return gen, nil
}
