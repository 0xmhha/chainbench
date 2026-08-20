package engine

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// GenesisSource produces a network's genesis bytes for a chain plugin, sized to
// the active validator count. It is a seam so BuildEnv does not depend on where
// the validator set and RLP extra-data come from: a baked preset today, and a
// live key deriver (once a Go extra-data encoder exists) later.
type GenesisSource interface {
	// Genesis builds the genesis.json for plugin using the first `validators`
	// preset entries (validators<=0 uses the whole set).
	Genesis(ctx context.Context, plugin registry.ChainPlugin, validators int) ([]byte, error)
}

// PresetGenesisSource builds genesis from a baked preset key set. The preset is
// the only source of a wbft-family network's RLP extra-data: the consensus
// family substitutes the extra-data into its template but does not compute it,
// so a freshly generated random key set cannot seed a valid wbft genesis on its
// own. Node identity/account keys (nodekey, unlock) come from the key registry;
// the validator set and its extra-data come from here.
type PresetGenesisSource struct {
	// KeysDir holds the preset metadata.json (validator addresses, BLS public
	// keys, and RLP extra-data).
	KeysDir string
	// ChainID, when non-zero, overrides the manifest chain id in the built
	// genesis (the custom-chain-id seam; the DSL env layer sets it).
	ChainID int64
	// ConfigOverrides sets keys in the genesis `config` object after the build —
	// the delayed-fork seam (e.g. {"bohoBlock":"10"} from
	// `setup --set genesis.overrides.*`). Empty applies none.
	ConfigOverrides map[string]string
	// Overlay is a JSON genesis fragment deep-merged into the built genesis
	// (extra alloc or system-contract params, from `setup --genesis-overlay`).
	// Empty applies none.
	Overlay []byte
}

// Genesis loads the preset, takes the first `validators` entries, and delegates
// to the chain's consensus family to build the genesis, applying any launch
// config overrides and overlay. ctx is accepted for future remote preset
// sources; the local metadata read does not use it.
func (s PresetGenesisSource) Genesis(_ context.Context, plugin registry.ChainPlugin, validators int) ([]byte, error) {
	preset, err := keyring.LoadPreset(s.KeysDir)
	if err != nil {
		return nil, fmt.Errorf("engine: genesis source: %w", err)
	}
	sub := preset.Take(validators)
	gen, err := genesis.BuildNetwork(plugin, genesis.Inputs{
		Validators: sub.Validators,
		BLSKeys:    sub.BLSKeys,
		ExtraData:  sub.ExtraData,
		Members:    sub.Members,
		Alloc:      sub.Alloc,
		ChainID:    s.ChainID,
	}, genesis.NetworkOptions{ConfigOverrides: s.ConfigOverrides, Overlay: s.Overlay})
	if err != nil {
		return nil, fmt.Errorf("engine: genesis source: %w", err)
	}
	return gen, nil
}
