package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
)

func TestRegistryBindingRefusesWrongTypesAndUnknownFields(t *testing.T) {
	ctx := context.Background()
	if _, err := invokeFeature[int, string](ctx, app.Deps{}, "node.status", 1); err == nil || !strings.Contains(err.Error(), "input") {
		t.Fatal("wrong scalar input reached a node operation")
	}
	if _, err := invokeFeatureFields[int](ctx, app.Deps{}, "chain.list", nil); err == nil || !strings.Contains(err.Error(), "output") {
		t.Fatal("wrong output contract was accepted")
	}
	if _, err := invokeFeatureFields[string](ctx, app.Deps{}, "node.status", map[string]any{"Missing": "value"}); err == nil || !strings.Contains(err.Error(), "no input field") {
		t.Fatal("unknown field was discarded")
	}
	if _, err := invokeFeatureFields[string](ctx, app.Deps{}, "node.status", map[string]any{"RPC": 123}); err == nil || !strings.Contains(err.Error(), "incompatible type") {
		t.Fatal("field type was coerced")
	}
	if _, err := invokeFeatureFields[string](ctx, app.Deps{}, "missing.operation", nil); err == nil {
		t.Fatal("missing operation was accepted")
	}
}

func TestRegistrySchemasRetainSpecificConstraintsAndStayStable(t *testing.T) {
	tool := Tool{Name: "scope", InputSchema: map[string]any{"type": "object", "properties": map[string]any{
		"rpc":          map[string]any{"type": "string", "description": "Requires an archive endpoint", "enum": []string{"archive"}},
		"workspaceDir": map[string]any{"type": "string", "pattern": "^/owned/"},
	}, "required": []string{"rpc"}}}
	tool = registrySchemas(tool)
	props := tool.InputSchema["properties"].(map[string]any)
	description := props["rpc"].(map[string]any)["description"].(string)
	tool = registrySchemas(tool)
	if props["rpc"].(map[string]any)["description"] != description || !strings.Contains(description, "Requires an archive endpoint") {
		t.Fatal("specific description lost or repeated on registration")
	}
	if props["rpc"].(map[string]any)["enum"].([]string)[0] != "archive" || props["workspaceDir"].(map[string]any)["pattern"] != "^/owned/" {
		t.Fatal("tool-specific constraint lost")
	}
	if len(tool.InputSchema["required"].([]string)) != 1 {
		t.Fatal("required alternatives were changed")
	}
	array := map[string]any{"type": "array", "items": map[string]any{"type": "string"}}
	other := registrySchemas(Tool{InputSchema: map[string]any{"properties": map[string]any{"rpc": array}}})
	if other.InputSchema["properties"].(map[string]any)["rpc"].(map[string]any)["type"] != "array" {
		t.Fatal("compound RPC input was flattened")
	}
}
