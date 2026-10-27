package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// ChainPresetInfo carries the declaration rather than a second Web model.
type ChainPresetInfo struct {
	ID          string          `json:"id"`
	Chain       string          `json:"chain"`
	Description string          `json:"description"`
	Content     json.RawMessage `json:"content"`
}

// ChainPresetSkip is a preset file the catalogue left out, and why.
type ChainPresetSkip struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

// ChainPresetCatalogueOut is the preset directory as the Web reads it: the
// presets a composition may start from, and the files it could not use.
type ChainPresetCatalogueOut struct {
	Items   []ChainPresetInfo `json:"items"`
	Skipped []ChainPresetSkip `json:"skipped"`
}

// ChainPresets reads the existing preset directory. Invalid and unattached
// composition choices are excluded; I/O errors are returned, never hidden.
func ChainPresets(root string) ([]ChainPresetInfo, error) {
	out, err := ChainPresetCatalogue(root)
	return out.Items, err
}

// ChainPresetCatalogue reads the preset directory and names each JSON file it
// leaves out with the reason, so a preset never disappears silently.
func ChainPresetCatalogue(root string) (ChainPresetCatalogueOut, error) {
	out := ChainPresetCatalogueOut{Items: []ChainPresetInfo{}, Skipped: []ChainPresetSkip{}}
	entries, err := os.ReadDir(root)
	if err != nil {
		return out, err
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		skip := func(reason string) {
			out.Skipped = append(out.Skipped, ChainPresetSkip{File: entry.Name(), Reason: reason})
		}
		raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			return out, err
		}
		env, err := dsl.ParseChainPreset(raw)
		if err != nil {
			skip(err.Error())
			continue
		}
		if _, err := registry.Get(env.Chain); err != nil {
			skip(err.Error())
			continue
		}
		if env.Attach != nil {
			skip("attaches to a running network; the Web composes its own")
			continue
		}
		if err := testengine.ValidateChainPreset(raw); err != nil {
			skip(err.Error())
			continue
		}
		if seen[env.ID] {
			return out, fmt.Errorf("preset: duplicate id %s", env.ID)
		}
		seen[env.ID] = true
		out.Items = append(out.Items, ChainPresetInfo{ID: env.ID, Chain: env.Chain, Description: env.Description, Content: raw})
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].ID < out.Items[j].ID })
	return out, nil
}

// ValidateChainPreset is the common usecase for a Web composition declaration.
func ValidateChainPreset(raw []byte) error { return testengine.ValidateChainPreset(raw) }

// ChainPresetContract enriches the parser-owned grammar with the selected
// manifest's launch vocabulary. It never exposes a raw argv escape hatch.
func ChainPresetContract(chain string) (map[string]any, error) {
	plugin, err := registry.Get(chain)
	if err != nil {
		return nil, err
	}
	dialect, err := nodeconfig.DialectFor(plugin.Manifest().Dialect)
	if err != nil {
		return nil, err
	}
	var schema map[string]any
	if err := json.Unmarshal(dsl.SchemaV2, &schema); err != nil {
		return nil, err
	}
	defs := schema["$defs"].(map[string]any)
	env := defs["envSpec"].(map[string]any)
	props := env["properties"].(map[string]any)
	props["chain"] = map[string]any{"const": chain}
	props["topology"] = map[string]any{"oneOf": []any{
		map[string]any{"type": "object", "title": "Node counts", "additionalProperties": false, "properties": map[string]any{
			"bp":       map[string]any{"type": "integer", "minimum": 1, "default": 4, "title": "블록 생산자"},
			"en":       map[string]any{"type": "integer", "minimum": 0, "default": 0, "title": "엔드포인트"},
			"pn":       map[string]any{"type": "integer", "minimum": 0, "default": 0, "title": "프록시"},
			"syncMode": map[string]any{"type": "string", "enum": []string{"full", "snap", "archive"}, "default": "full", "title": "동기화 방식"},
		}},
		map[string]any{"type": "object", "title": "Node table", "additionalProperties": false, "required": []string{"nodes"}, "properties": map[string]any{
			"nodes": map[string]any{"type": "array", "items": map[string]any{"type": "object", "additionalProperties": false, "required": []string{"role"}, "properties": map[string]any{
				"role": map[string]any{"type": "string", "enum": []string{"bp", "en", "pn"}}, "binary": map[string]any{"type": "string"},
				"sync": map[string]any{"type": "string", "enum": []string{"full", "snap", "archive"}}, "bootnode": map[string]any{"type": "boolean"},
				"index": map[string]any{"type": "integer", "minimum": 1}, "config": map[string]any{"type": "string"}, "key": map[string]any{"type": "string"},
			}}},
		}},
	}}
	group := map[string]any{"type": "object", "additionalProperties": false, "properties": dialect.OptionSchemas(false)}
	nodeGroup := map[string]any{"type": "object", "additionalProperties": false, "properties": dialect.OptionSchemas(true)}
	props["launch"] = map[string]any{"type": "object", "properties": map[string]any{"all": group, "bp": group, "en": group, "pn": group}, "additionalProperties": nodeGroup, "propertyNames": map[string]any{"pattern": "^(all|bp|en|pn|node[1-9][0-9]*)$"}}
	configProps := nodeconfig.ConfigOptionSchemas()
	props["config"] = map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "object", "additionalProperties": false, "properties": configProps}, "propertyNames": map[string]any{"pattern": "^(all|bp|en|pn|node[1-9][0-9]*)$"}}
	return schema, nil
}
