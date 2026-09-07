package app

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/blueprint"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// BlueprintFromPresetIn names the key set to describe and the network to
// describe it as.
type BlueprintFromPresetIn struct {
	// KeysDir is the key set the document is written from.
	KeysDir string
	// Chain names a registered chain, or Manifest an external one.
	Chain    string
	Manifest string
	// Producers and Endpoints size the network; zero producers means every
	// identity the set declares as a validator.
	Producers int
	Endpoints int
	// Binary is the node executable to record, when the caller knows it.
	Binary string
	// Peering is the peer graph to record; empty leaves the default.
	Peering string
}

// BlueprintFromPresetOut is the document, ready to write.
type BlueprintFromPresetOut struct {
	// YAML is the document as it should be saved.
	YAML []byte
	// Nodes is how many nodes it describes, for a surface to report.
	Nodes int
}

// BlueprintFromPreset writes out the network a key set would compose.
//
// It is the visible half of the inversion: a composition used to go from a
// preset straight to a network with nothing inspectable in between, and it now
// goes through a document an operator can read, edit and commit. What comes
// back is bytes rather than a file, because where it lands is the surface's
// question — stdout for a pipeline, a path for a repository.
func BlueprintFromPreset(ctx context.Context, _ Deps, in BlueprintFromPresetIn) (BlueprintFromPresetOut, error) {
	if in.KeysDir == "" {
		return BlueprintFromPresetOut{}, fmt.Errorf("app: blueprint: name the key set to describe")
	}
	// Ensure rather than a bare load, so this reads a set the same way the
	// composition does and reports the same problem if it cannot.
	set, err := store.PresetKeys{Path: in.KeysDir}.Ensure(ctx, in.Producers+in.Endpoints)
	if err != nil {
		return BlueprintFromPresetOut{}, err
	}
	bp, err := blueprint.FromPreset(set, blueprint.FromPresetIn{
		Dir: in.KeysDir, Chain: in.Chain, Manifest: in.Manifest,
		Producers: in.Producers, Endpoints: in.Endpoints,
		Binary: in.Binary, Peering: in.Peering,
	})
	if err != nil {
		return BlueprintFromPresetOut{}, err
	}
	raw, err := blueprint.Marshal(bp)
	if err != nil {
		return BlueprintFromPresetOut{}, err
	}
	return BlueprintFromPresetOut{YAML: raw, Nodes: len(bp.Nodes)}, nil
}
