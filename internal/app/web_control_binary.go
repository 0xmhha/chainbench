package app

import (
	"context"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// bindWebControlBinary accepts only the reviewed executable or the engine's
// native-name test copy. Matching bytes at arbitrary paths are not sufficient.
// A selected named per-node executable has no reviewed asset binding yet and
// cannot borrow the base binary verification to authorize start or stop.
func bindWebControlBinary(ctx context.Context, state State, p webChainPayload, lookup resource.Lookup) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	for _, ns := range state.Nodes {
		for _, id := range p.Input.NodeIDs {
			if string(ns.NodeLabel()) == id && ns.Binary != "" {
				return "", ErrDeploymentConflict
			}
		}
	}
	if state.Binary == "" || len(p.Binary.SHA256) != 64 {
		return "", ErrDeploymentConflict
	}
	allowed := state.Binary == p.ExecutionBinary && p.ExecutionBinary != ""
	if !allowed {
		plugin, err := registry.Get(p.Binary.Chain)
		if err != nil {
			return "", ErrDeploymentConflict
		}
		wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
		if err != nil {
			return "", err
		}
		native, err := wc.Resolve(resource.PurposeBinaries, p.Binary.SHA256+"/"+plugin.Manifest().Binary)
		if err != nil {
			return "", err
		}
		allowed = filepath.Clean(state.Binary) == filepath.Clean(native)
	}
	if !allowed {
		return "", ErrDeploymentConflict
	}
	access, err := (resource.Opener{Lookup: lookup}).Open(state.Target)
	if err != nil {
		return "", err
	}
	checksum, err := access.Files.Checksum(ctx, state.Binary)
	if err != nil {
		return "", err
	}
	if checksum != "sha256:"+p.Binary.SHA256 {
		return "", ErrDeploymentConflict
	}
	return state.Binary, nil
}
