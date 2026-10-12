package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/netip"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"go.yaml.in/yaml/v3"
)

// webConfigAsset is a registered configuration file a node replacement
// applies: its checksum-pinned asset and the engine config values it holds.
type webConfigAsset struct {
	Asset     WebAsset          `json:"asset"`
	Overrides map[string]string `json:"overrides"`
}

// webConfigAssetOverrides reads a registered configuration file as the
// engine's node config values. Only knobs the engine renders are accepted;
// syncMode takes the engine's choices and a host must be an IP address.
func webConfigAssetOverrides(name string, data []byte) (map[string]string, error) {
	var value map[string]any
	switch strings.ToLower(filepath.Ext(name)) {
	case ".json":
		dec := json.NewDecoder(bytes.NewReader(data))
		if dec.Decode(&value) != nil || dec.Decode(new(any)) != io.EOF {
			return nil, errors.New("a configuration file holds one JSON object")
		}
	case ".yaml", ".yml":
		dec := yaml.NewDecoder(bytes.NewReader(data))
		if dec.Decode(&value) != nil || dec.Decode(new(any)) != io.EOF {
			return nil, errors.New("a configuration file holds one YAML object")
		}
	default:
		return nil, errors.New("a configuration file is JSON or YAML")
	}
	if len(value) == 0 {
		return nil, errors.New("the configuration file sets no node config value")
	}
	schema := nodeconfig.ConfigOptionSchemas()
	out := map[string]string{}
	for key, raw := range value {
		field, ok := schema[key]
		if !ok {
			return nil, errors.New("the configuration file sets " + key + ", which the engine does not render")
		}
		text, ok := raw.(string)
		if !ok {
			return nil, errors.New(key + " must be a string")
		}
		if key == "syncMode" {
			if !slices.Contains(field.(map[string]any)["enum"].([]string), text) {
				return nil, errors.New("syncMode must be one of the engine's choices")
			}
		} else if _, err := netip.ParseAddr(text); err != nil {
			return nil, errors.New(key + " must be an IP address")
		}
		out[key] = text
	}
	return out, nil
}

// prepareConfigAsset pins the registered configuration file a node.swap
// selected and the values it holds.
func (e *WebChainEngine) prepareConfigAsset(ctx context.Context, p *webChainPayload) error {
	if p.Arguments.ConfigAssetID == "" {
		return nil
	}
	if p.Input.Operation != "node.swap" || p.Arguments.ConfigOverrides != nil {
		return errors.New("a registered configuration file replaces a node's config alone, without typed choices")
	}
	if e.manifests == nil {
		return ErrDeploymentNotFound
	}
	asset, err := e.manifests.Asset(p.Arguments.ConfigAssetID)
	if err != nil {
		return err
	}
	if asset.Kind != "configuration" {
		return errors.New("select a registered configuration file")
	}
	path, err := e.manifests.assets.MaterializeFile(ctx, asset.ID, asset.Checksum, manifestHash([]byte("config-asset:"+asset.ID+":"+asset.Checksum)))
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	overrides, err := webConfigAssetOverrides(asset.Name, data)
	if err != nil {
		return err
	}
	p.ConfigAsset = &webConfigAsset{Asset: asset, Overrides: overrides}
	return nil
}
