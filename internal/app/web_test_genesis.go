package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/dsl"
)

func (e *WebChainEngine) projectTestGenesis(ctx context.Context, p webChainPayload, run webTestRun) ([]json.RawMessage, []webPresetGenesis, error) {
	content := make([]json.RawMessage, 0, len(run.Cases))
	genesis := []webPresetGenesis{}
	pinned, err := json.Marshal(struct {
		Cases       []webExecutableCase
		Set, Config DeploymentDocument
		Binary      ManifestBinaryEvidence
		Keys        webKeySnapshot
	}{run.Cases, p.Set, p.Config, p.Binary, p.Keys})
	if err != nil {
		return nil, nil, err
	}
	fingerprint := manifestHash(pinned)
	seen := map[string]bool{}
	for _, c := range run.Cases {
		if err := validateWebDocumentAssetRefs(c.Document.DeploymentDocumentInput); err != nil {
			return nil, nil, err
		}
		spec, err := dsl.Parse(c.Content)
		if err != nil {
			return nil, nil, err
		}
		raw := append(json.RawMessage(nil), c.Content...)
		if spec.Chain.GenesisExisting != "" {
			id, err := webAssetRefID(spec.Chain.GenesisExisting)
			if err != nil {
				return nil, nil, err
			}
			if e.manifests == nil {
				return nil, nil, ErrDeploymentNotFound
			}
			asset, err := e.manifests.Asset(id)
			if err != nil {
				return nil, nil, err
			}
			if asset.Kind != "template" || asset.Compatibility["format"] != "json" {
				return nil, nil, errors.New("finished test genesis requires a registered JSON template asset")
			}
			path, err := e.manifests.assets.MaterializeFile(ctx, id, asset.Checksum, fingerprint)
			if err != nil {
				return nil, nil, fmt.Errorf("%w: registered test genesis changed or is unavailable", ErrDeploymentConflict)
			}
			bytes, err := os.ReadFile(path)
			var header struct {
				Config map[string]json.RawMessage `json:"config"`
			}
			if err != nil || manifestHash(bytes) != asset.Checksum {
				return nil, nil, ErrDeploymentConflict
			}
			if json.Unmarshal(bytes, &header) != nil || header.Config["chainId"] == nil {
				return nil, nil, errors.New("finished test genesis requires config.chainId")
			}
			raw, err = projectWebCaseGenesisPath(raw, path)
			if err != nil {
				return nil, nil, err
			}
			if !seen[id] {
				genesis = append(genesis, webPresetGenesis{Asset: asset, Fingerprint: fingerprint, Path: path})
				seen[id] = true
			}
		}
		content = append(content, raw)
	}
	return content, genesis, ctx.Err()
}

func (e *WebChainEngine) recheckTestGenesis(ctx context.Context, p webChainPayload) error {
	content, genesis, err := e.projectTestGenesis(ctx, p, *p.TestRun)
	if err != nil {
		return err
	}
	before, err := json.Marshal(struct {
		Content []json.RawMessage
		Genesis []webPresetGenesis `json:"genesis,omitempty"`
	}{p.TestRun.Content, p.TestRun.Genesis})
	if err != nil {
		return err
	}
	after, err := json.Marshal(struct {
		Content []json.RawMessage
		Genesis []webPresetGenesis `json:"genesis,omitempty"`
	}{content, genesis})
	if err != nil {
		return err
	}
	if string(before) != string(after) {
		return ErrDeploymentConflict
	}
	return nil
}

func projectWebCaseGenesisPath(raw json.RawMessage, path string) (json.RawMessage, error) {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	var version string
	_ = json.Unmarshal(root["schemaVersion"], &version)
	field := "chain"
	if version == "2" {
		field = "chainPreset"
	}
	var env map[string]json.RawMessage
	if err := json.Unmarshal(root[field], &env); err != nil {
		return nil, err
	}
	if version == "2" {
		var genesis map[string]json.RawMessage
		if err := json.Unmarshal(env["genesis"], &genesis); err != nil {
			return nil, err
		}
		genesis["ref"], _ = json.Marshal(path)
		env["genesis"], _ = json.Marshal(genesis)
	} else {
		env["genesisExisting"], _ = json.Marshal(path)
	}
	root[field], _ = json.Marshal(env)
	return json.Marshal(root)
}
