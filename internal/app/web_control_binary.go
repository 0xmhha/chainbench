package app

import (
	"context"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// bindWebControlBinary accepts only the reviewed executable or the engine's
// native-name test copy. Matching bytes at arbitrary paths are not sufficient.
// Named per-node executables require their own reviewed asset binding; they
// cannot borrow the base binary verification to authorize start or stop.
func bindWebControlBinary(ctx context.Context, state State, p webChainPayload, lookup resource.Lookup) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	for _, ns := range state.Nodes {
		for _, id := range p.Input.NodeIDs {
			if string(ns.NodeLabel()) == id && ns.Binary != "" {
				if !webNodeBinaryDeclarationsMatch(state, ns) {
					return "", ErrDeploymentConflict
				}
				binding := p.CurrentNodeBinary
				if binding == nil || binding.NodeID != id || binding.Path != state.Binaries[ns.Binary] {
					return "", ErrDeploymentConflict
				}
				expected, err := webNodeBinaryTarget(p, binding.Evidence)
				if err != nil || expected != binding.Path {
					return "", ErrDeploymentConflict
				}
				access, err := (resource.Opener{Lookup: lookup}).Open(webNodeTarget(state, ns))
				if err != nil {
					return "", err
				}
				if err = access.VerifyRegularFile(ctx, binding.Path); err != nil {
					return "", ErrDeploymentConflict
				}
				if webControlNeedsExecutable(p.Input.Operation) {
					if err = access.VerifyExecutable(ctx, binding.Path); err != nil {
						return "", ErrDeploymentConflict
					}
				}
				checksum, err := access.Files.Checksum(ctx, binding.Path)
				if err != nil || checksum != "sha256:"+binding.Evidence.SHA256 {
					return "", ErrDeploymentConflict
				}
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
	if webControlNeedsExecutable(p.Input.Operation) && p.CurrentNodeBinary == nil {
		if err = access.VerifyExecutable(ctx, state.Binary); err != nil {
			return "", ErrDeploymentConflict
		}
	}
	return state.Binary, nil
}

func webControlNeedsExecutable(operation string) bool {
	switch operation {
	case "node.start", "node.restart", "node.swap", "node.reset":
		return true
	default:
		return false
	}
}
