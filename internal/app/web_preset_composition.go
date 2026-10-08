package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testengine"
)

type webPresetComposition struct {
	Document DeploymentDocument `json:"document"`
	Request  ChainUpIn          `json:"request"`
	Plan     ComposePlan        `json:"plan"`
	Genesis  *webPresetGenesis  `json:"genesis,omitempty"`
}

type webPresetGenesis struct {
	Asset       WebAsset `json:"asset"`
	Fingerprint string   `json:"fingerprint"`
	Path        string   `json:"path"`
}

func (e *WebChainEngine) preparePresetComposition(ctx context.Context, p *webChainPayload) ([]resource.Request, error) {
	ref := p.Arguments.ChainPresetRef
	if ref == nil || ref.Revision < 1 {
		return nil, errors.New("select a saved chain preset with an explicit revision")
	}
	doc, err := e.documents.DocumentRevision(ref.ID, ref.Revision)
	if err != nil {
		return nil, err
	}
	latest, err := e.documents.DocumentRevision(ref.ID, 0)
	if err != nil {
		return nil, err
	}
	if latest.Revision != ref.Revision {
		return nil, ErrDeploymentConflict
	}
	if doc.Kind != "chain-preset" {
		return nil, errors.New("selected document is not a chain preset")
	}
	p.Keys, err = e.pinKeys(ctx)
	if err != nil {
		return nil, err
	}
	p.Preset = &webPresetComposition{Document: doc}
	projection, err := e.projectPresetComposition(ctx, *p)
	if err != nil {
		return nil, err
	}
	p.Preset = &projection
	p.Arguments.Validators = projection.Plan.Nodes.BP
	return webPresetRequests(projection.Request), nil
}

func (e *WebChainEngine) projectPresetComposition(ctx context.Context, p webChainPayload) (webPresetComposition, error) {
	var out webPresetComposition
	if p.Preset == nil {
		return out, ErrDeploymentConflict
	}
	doc := p.Preset.Document
	env, err := dsl.ParseChainPreset(doc.Content)
	if err != nil {
		return out, err
	}
	if env.Chain != p.Binary.Chain {
		return out, errors.New("selected preset belongs to a different chain")
	}
	if p.Manifest.Source == "external" || env.Attach != nil || env.Blueprint != "" || env.Manifest != "" || env.GenesisTemplate != "" || env.Upgrade != nil || len(env.Accounts) > 0 || env.Genesis != nil && len(env.Genesis.PerBinary) > 0 {
		return out, errors.New("preset file references, attachments, declared accounts and upgrades require their own pinned asset contracts")
	}
	if err := validateWebDocumentAssetRefs(doc.DeploymentDocumentInput); err != nil {
		return out, err
	}
	var genesis *WebAsset
	if env.Genesis != nil && env.Genesis.Ref != "" {
		id, err := webAssetRefID(env.Genesis.Ref)
		if err != nil {
			return out, err
		}
		if e.manifests == nil {
			return out, ErrDeploymentNotFound
		}
		asset, err := e.manifests.Asset(id)
		if err != nil {
			return out, err
		}
		if asset.Kind != "template" || asset.Compatibility["format"] != "json" {
			return out, errors.New("finished genesis requires a registered JSON template asset")
		}
		genesis = &asset
	}
	if len(env.Binaries) > 1 {
		return out, errors.New("preset mixed binaries require a verified asset for every binary")
	}
	for name, binary := range env.Binaries {
		if name != dsl.BinaryDefault || binary.Chain != "" && binary.Chain != env.Chain {
			return out, errors.New("only the reviewed default preset binary is registered")
		}
	}
	if env.Keys != nil && env.Keys.NodeKeys != nil {
		keys := env.Keys.NodeKeys
		if keys.Source != "" && keys.Source != "keyPreset" || keys.Ref != "" && keys.Ref != "presets/keys" || keys.Validators != 0 {
			return out, errors.New("preset keys must use the reviewed key snapshot")
		}
	}
	for _, knobs := range env.Launch {
		for key := range knobs {
			if webManagedLaunchInput(key) {
				return out, errors.New("preset paths and ports require resolved resource or asset references")
			}
		}
	}
	pinned, err := json.Marshal(struct {
		Document, Set, Config DeploymentDocument
		Binary                ManifestBinaryEvidence
		Keys                  webKeySnapshot
		Genesis               *WebAsset
	}{doc, p.Set, p.Config, p.Binary, p.Keys, genesis})
	if err != nil {
		return out, err
	}
	dir, err := filepath.Abs(filepath.Join(e.root, "preset-inputs", manifestHash(pinned)))
	if err != nil {
		return out, err
	}
	if err = writeWebDeploymentInputs(ctx, dir, p.Set, p.Config); err != nil {
		return out, err
	}
	content := doc.Content
	var pinnedGenesis *webPresetGenesis
	if genesis != nil {
		fingerprint := manifestHash(pinned)
		path, err := e.manifests.assets.MaterializeFile(ctx, genesis.ID, genesis.Checksum, fingerprint)
		if err != nil {
			return out, errors.New("registered genesis input changed or is unavailable")
		}
		raw, err := os.ReadFile(path)
		if err != nil || manifestHash(raw) != genesis.Checksum {
			return out, ErrDeploymentConflict
		}
		var header struct {
			Config map[string]json.RawMessage `json:"config"`
		}
		if json.Unmarshal(raw, &header) != nil || header.Config["chainId"] == nil {
			return out, errors.New("finished genesis requires a JSON object with config.chainId")
		}
		var projection map[string]json.RawMessage
		if err = json.Unmarshal(content, &projection); err != nil {
			return out, err
		}
		var declaration map[string]json.RawMessage
		if err = json.Unmarshal(projection["genesis"], &declaration); err != nil {
			return out, err
		}
		declaration["ref"], _ = json.Marshal(path)
		projection["genesis"], _ = json.Marshal(declaration)
		content, err = json.Marshal(projection)
		if err != nil {
			return out, err
		}
		pinnedGenesis = &webPresetGenesis{Asset: *genesis, Fingerprint: fingerprint, Path: path}
	}
	in := RunSuiteIn{DataDir: filepath.Join(dir, "planning"), Chain: p.Binary.Chain, Binary: p.ExecutionBinary, BinaryOverrides: map[string]string{dsl.BinaryDefault: p.ExecutionBinary}, KeysDir: webAcceptedKeyPath(e.root, p.Keys.SHA256), Server: resource.ServerRef{SetPath: filepath.Join(dir, "server-set.yaml"), Name: p.Arguments.ServerRef}, WorkspaceConfigPath: filepath.Join(dir, "workspace-config.yaml")}
	request, plan, err := testengine.PlanChainPreset(ctx, content, in)
	if err != nil {
		return out, err
	}
	total := plan.Nodes.BP + plan.Nodes.EN + plan.Nodes.PN
	if plan.Nodes.AutoSize || plan.Nodes.BP < 1 || total < 1 || total > 128 {
		return out, errors.New("preset layout must resolve between one and 128 nodes with producers")
	}
	if err = validateWebTableInputs(request); err != nil {
		return out, err
	}
	request.DataDir = p.ControlDir
	plan.Workspace = p.ControlDir
	if _, err = chainsetup.GenesisOptsFor(ChainGenesisIn{GenesisExisting: request.GenesisExisting, ChainID: request.ChainID, Set: request.GenesisSet, OverlayPath: request.OverlayPath}); err != nil {
		return out, err
	}
	return webPresetComposition{Document: doc, Request: request, Plan: plan, Genesis: pinnedGenesis}, nil
}

func validateWebTableInputs(in ChainUpIn) error {
	if in.Topology != nil {
		for _, n := range in.Topology.Nodes {
			if n.Config != "" || n.Key != "" || n.Binary != "" && n.Binary != dsl.BinaryDefault {
				return errors.New("per-node files, keys and named binaries require registered immutable assets")
			}
		}
	}
	return nil
}

func webPresetRequests(in ChainUpIn) []resource.Request {
	requests := []resource.Request{}
	if in.Topology != nil {
		for _, n := range in.Topology.Sorted() {
			requests = append(requests, resource.Request{Role: n.NodeRole(), Label: node.LabelFor(n.Index)})
		}
		return requests
	}
	for _, group := range []struct {
		role  node.Role
		count int
	}{{node.RoleBP, in.BPCount}, {node.RoleEN, in.ENCount}, {node.RolePN, in.PNCount}} {
		for range group.count {
			requests = append(requests, resource.Request{Role: group.role})
		}
	}
	return requests
}

func webManagedLaunchInput(key string) bool {
	switch key {
	case "datadir", "datadir.ancient", "config", "nodekey", "keystore", "password", "ipcpath", "txpool.journal", "authrpc.jwtsecret", "port", "discovery.port", "http.port", "ws.port", "authrpc.port", "metrics.port":
		return true
	}
	return false
}
