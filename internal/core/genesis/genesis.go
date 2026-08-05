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
	return p.Family().BuildGenesis(tmpl, registry.GenesisParams{
		ChainID:    m.ChainID,
		Validators: in.Validators,
		BLSKeys:    in.BLSKeys,
		ExtraData:  in.ExtraData,
		Members:    in.Members,
		Alloc:      in.Alloc,
		Coinbase:   in.Coinbase,
	})
}
