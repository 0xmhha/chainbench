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

// ChainPresets reads the existing preset directory. Invalid and unattached
// composition choices are excluded; I/O errors are returned, never hidden.
func ChainPresets(root string) ([]ChainPresetInfo, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := []ChainPresetInfo{}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(root, entry.Name()))
		if err != nil {
			return nil, err
		}
		env, err := dsl.ParseChainPreset(raw)
		if err != nil {
			continue
		}
		if _, err := registry.Get(env.Chain); err != nil {
			continue
		}
		if env.Attach != nil || testengine.ValidateChainPreset(raw) != nil {
			continue
		}
		if seen[env.ID] {
			return nil, fmt.Errorf("preset: duplicate id %s", env.ID)
		}
		seen[env.ID] = true
		out = append(out, ChainPresetInfo{ID: env.ID, Chain: env.Chain, Description: env.Description, Content: raw})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
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
			"bp":       map[string]any{"type": "integer", "minimum": 1, "default": 4},
			"en":       map[string]any{"type": "integer", "minimum": 0, "default": 0},
			"pn":       map[string]any{"type": "integer", "minimum": 0, "default": 0},
			"syncMode": map[string]any{"type": "string", "enum": []string{"full", "snap", "archive"}, "default": "full"},
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
