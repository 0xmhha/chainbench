// Package genesis builds a chain's genesis.json by dispatching on its consensus
// family: the wbft family (stablenet + wbft) substitutes the chain's embedded
// template with the validator set and chain id; the poa family (wemix) has a
// deploy-time genesis and is handled by its bootstrap in G7. This is the seam
// the setup pipeline calls to produce genesis bytes before provisioning
// (docs/CHAINBENCH_GO_REDESIGN.md §3, §8).
package genesis

import (
	"errors"
	"fmt"

	"github.com/0xmhha/chainbench/pkg/consensus/poa"
	"github.com/0xmhha/chainbench/pkg/consensus/wbft"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

// ErrNoStaticGenesis is returned for chains whose genesis is not built from a
// static template (the poa/registry family bootstraps its genesis at deploy
// time).
var ErrNoStaticGenesis = errors.New("genesis: chain has no static template")

// Inputs are the per-network genesis values the setup phase supplies (from the
// preset key metadata): the validator set and the RLP extra-data that encodes
// it.
type Inputs struct {
	Validators []string // validator addresses (0x-hex) — wbft family
	BLSKeys    []string // BLS public keys (0x-hex), aligned with Validators
	ExtraData  string   // validator extra-data (0x-hex) — wbft family
	Members    []string // governance council addresses (0x-hex) — anzeon system contracts
	Coinbase   string   // block coinbase (0x-hex) — poa family; default zero
}

// Build produces the genesis.json bytes for a chain plugin. The chain id comes
// from the manifest; validators/BLS/extraData come from in.
func Build(p registry.ChainPlugin, in Inputs) ([]byte, error) {
	m := p.Manifest()
	switch m.ConsensusFamily {
	case "wbft":
		tmpl := p.GenesisTemplate()
		if len(tmpl) == 0 {
			return nil, fmt.Errorf("genesis: chain %q (wbft) has no template: %w", m.ID, ErrNoStaticGenesis)
		}
		return wbft.BuildGenesis(tmpl, wbft.GenesisParams{
			ChainID:    m.ChainID,
			Validators: in.Validators,
			BLSKeys:    in.BLSKeys,
			ExtraData:  in.ExtraData,
			Members:    in.Members,
		})
	case "poa":
		tmpl := p.GenesisTemplate()
		if len(tmpl) == 0 {
			return nil, fmt.Errorf("genesis: chain %q (poa) has no template: %w", m.ID, ErrNoStaticGenesis)
		}
		// The validator set is not in a poa genesis — membership is set at
		// bootstrap (governance + etcd, see poa.BootstrapPlan).
		return poa.BuildGenesis(tmpl, m.ChainID, in.Coinbase)
	default:
		return nil, fmt.Errorf("genesis: chain %q has unknown consensus family %q", m.ID, m.ConsensusFamily)
	}
}
