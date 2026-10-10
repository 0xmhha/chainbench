package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// observedWebBinary resolves the same private binding used by job acceptance.
// It exposes asset identity only after checking the source and deployed bytes.
func (e *WebChainEngine) observedWebBinary(ctx context.Context, state State, p webChainPayload, lookup resource.Lookup) (*webNodeBinary, error) {
	if state.ManifestPath == "" && state.TemplatePath == "" {
		plugin, err := registry.Get(state.Chain)
		if err != nil {
			return nil, err
		}
		p.Binary.Chain = plugin.Protocol().Name
	} else {
		if state.ManifestPath != filepath.Join(p.ControlDir, "selected-manifest.json") || state.TemplatePath != filepath.Join(p.ControlDir, "selected-template.json") || e.manifests == nil {
			return nil, ErrDeploymentConflict
		}
		manifest, err := os.ReadFile(state.ManifestPath)
		if err != nil {
			return nil, err
		}
		template, err := os.ReadFile(state.TemplatePath)
		if err != nil {
			return nil, err
		}
		items, err := e.manifests.List()
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if item.Source != "external" || !bytes.Equal(item.Manifest, manifest) || item.Template != string(template) {
				continue
			}
			plugin, err := ValidateManifest(item.ManifestInput)
			if err == nil && plugin.Manifest().ID == state.Chain {
				p.Binary.Chain = plugin.Protocol().Name
				break
			}
		}
		if p.Binary.Chain == "" {
			return nil, ErrDeploymentConflict
		}
	}
	set, err := deploymentSet(p.Set.DeploymentDocumentInput)
	if err != nil {
		return nil, err
	}
	for _, entry := range set.Servers {
		server, err := set.ByName(entry.Name)
		if err == nil && resource.TargetOf(server, state.Target.DataRoot) == state.Target {
			p.Arguments.ServerRef = server.Name
			break
		}
	}
	if p.Arguments.ServerRef == "" {
		return nil, ErrDeploymentConflict
	}
	binding, _, err := e.currentWebNodeBinary(ctx, state, p, lookup)
	return binding, err
}
