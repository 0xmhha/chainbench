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
					return "", webConflict("recorded node binary declarations differ")
				}
				binding := p.CurrentNodeBinary
				if binding == nil || binding.NodeID != id || binding.Path != state.Binaries[ns.Binary] {
					return "", webConflict("recorded node binary binding differs")
				}
				expected, err := webNodeBinaryTarget(p, binding.Evidence)
				if err != nil || expected != binding.Path {
					return "", webConflict("node binary is not at its registered path")
				}
				access, err := (resource.Opener{Lookup: lookup}).Open(webNodeTarget(state, ns))
				if err != nil {
					return "", err
				}
				if err = access.VerifyRegularFile(ctx, binding.Path); err != nil {
					return "", webConflict("node binary file is missing on its machine")
				}
				if webControlNeedsExecutable(p.Input.Operation) {
					if err = access.VerifyExecutable(ctx, binding.Path); err != nil {
						return "", webConflict("node binary is not executable on its machine")
					}
				}
				checksum, err := access.Files.Checksum(ctx, binding.Path)
				if err != nil || checksum != "sha256:"+binding.Evidence.SHA256 {
					return "", webConflict("node binary checksum differs on its machine")
				}
			}
		}
	}
	if state.Binary == "" || len(p.Binary.SHA256) != 64 {
		return "", webConflict("the recorded network binary or its reviewed checksum is missing")
	}
	allowed := state.Binary == p.ExecutionBinary && p.ExecutionBinary != ""
	if !allowed {
		plugin, err := registry.Get(p.Binary.Chain)
		if err != nil {
			return "", webConflict("the recorded chain has no registered plugin")
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
		return "", webConflict("the recorded network binary is not the reviewed binary")
	}
	// The executable is checked on the selected node's own machine, which for
	// a spread network need not be the network's first target.
	spec := state.Target
	for _, ns := range state.Nodes {
		if len(p.Input.NodeIDs) == 1 && string(ns.NodeLabel()) == p.Input.NodeIDs[0] {
			spec = webNodeTarget(state, ns)
		}
	}
	access, err := (resource.Opener{Lookup: lookup}).Open(spec)
	if err != nil {
		return "", err
	}
	checksum, err := access.Files.Checksum(ctx, state.Binary)
	if err != nil {
		return "", err
	}
	if checksum != "sha256:"+p.Binary.SHA256 {
		return "", webConflict("network binary checksum differs on the node machine")
	}
	if webControlNeedsExecutable(p.Input.Operation) && p.CurrentNodeBinary == nil {
		if err = access.VerifyExecutable(ctx, state.Binary); err != nil {
			return "", webConflict("network binary is not executable on the node machine")
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
