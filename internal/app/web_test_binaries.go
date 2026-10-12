package app

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/resource"
)

// webNamedBinary is one binary name a test declares besides default, bound to
// the registered asset that runs under it and where the job places it.
type webNamedBinary struct {
	Name            string                 `json:"name"`
	AssetID         string                 `json:"assetId"`
	Binary          ManifestBinaryEvidence `json:"binary"`
	ExecutionBinary string                 `json:"executionBinary"`
}

// declareTestBinaries records the binary names a case declares besides
// default and the chain each runs, refusing a name two cases give different
// chains.
func declareTestBinaries(declared map[string]string, spec dsl.Spec) error {
	for name := range spec.Chain.Binaries {
		if name == dsl.BinaryDefault {
			continue
		}
		chain := spec.Chain.BinaryChains[name]
		if chain == "" {
			chain = spec.Chain.Name
		}
		if known, ok := declared[name]; ok && known != chain {
			return fmt.Errorf("binary %q runs chain %s in one case and %s in another", name, known, chain)
		}
		declared[name] = chain
	}
	return nil
}

// pinNamedBinaries binds every declared binary name to its selected asset:
// verified against the chain the name runs, matched to the target platform,
// and placed under its checksum. A selection for a name no case declares is
// refused rather than carried along unused.
func (e *WebChainEngine) pinNamedBinaries(ctx context.Context, p webChainPayload, wc resource.WorkspaceConfig, declared map[string]string) ([]webNamedBinary, error) {
	for name := range p.Arguments.BinaryAssets {
		if _, ok := declared[name]; !ok {
			return nil, fmt.Errorf("binary %q is selected but no selected case declares it", name)
		}
	}
	names := make([]string, 0, len(declared))
	for name := range declared {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]webNamedBinary, 0, len(names))
	for _, name := range names {
		id := p.Arguments.BinaryAssets[name]
		if id == "" {
			return nil, fmt.Errorf("select a registered %s binary for %q", declared[name], name)
		}
		asset, err := e.binaryAsset(id)
		if err != nil {
			return nil, fmt.Errorf("binary %q: %w", name, err)
		}
		plugin, err := registry.Get(declared[name])
		if err != nil {
			return nil, err
		}
		evidence, err := e.verify(ctx, asset, declared[name])
		if err != nil {
			return nil, fmt.Errorf("binary %q: %w", name, err)
		}
		if evidence.OS != p.Binary.OS || evidence.Architecture != p.Binary.Architecture {
			return nil, fmt.Errorf("binary %q does not match the target platform", name)
		}
		path, err := wc.Resolve(resource.PurposeBinaries, evidence.SHA256+"/"+plugin.Manifest().Binary)
		if err != nil {
			return nil, err
		}
		out = append(out, webNamedBinary{Name: name, AssetID: id, Binary: evidence, ExecutionBinary: path})
	}
	return out, nil
}

// webExecutableClaims are the node executables a launching job runs on its
// target: the default and each named binary, one claim per file name.
func webExecutableClaims(p webChainPayload) []WebResourceClaim {
	seen := map[string]bool{}
	out := []WebResourceClaim{}
	for _, path := range append([]string{p.ExecutionBinary}, namedExecutionBinaries(p)...) {
		name := filepath.Base(path)
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, WebResourceClaim{HostIdentity: p.Target.HostIdentity, Executable: name})
	}
	return out
}

func namedExecutionBinaries(p webChainPayload) []string {
	out := make([]string, 0, len(p.NamedBinaries))
	for _, named := range p.NamedBinaries {
		out = append(out, named.ExecutionBinary)
	}
	return out
}
