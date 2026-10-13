package dsl_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

func TestPresetValuesParseAndMergeWithoutLosingTypes(t *testing.T) {
	base := []byte(`{"schemaVersion":"2","kind":"chain-preset","id":"base","chain":"wbft","values":{"count":2,"enabled":true,"recipient":"0xabc","payload":{"items":[1,2]},"removed":"old"}}`)
	env, err := dsl.ParseChainPreset(base)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	var wanted, got map[string]any
	_ = json.Unmarshal(base, &wanted)
	_ = json.Unmarshal(encoded, &got)
	if !equalJSON(t, wanted["values"], got["values"]) {
		t.Fatal("preset values disappeared")
	}
	input := []byte(`{"schemaVersion":"2","kind":"case","id":"c","chainPreset":{"extends":"base","values":{"count":3,"payload":{"items":[9]},"removed":null}},"steps":[{"expect":"blockNumber","is":"$count"}]}`)
	merged, err := dsl.InlineChainPreset(input, func(id string) ([]byte, error) { return base, nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err = dsl.Parse(merged); err != nil {
		t.Fatal(err)
	}
	_ = json.Unmarshal(merged, &got)
	values := got["chainPreset"].(map[string]any)["values"].(map[string]any)
	if values["count"] != float64(3) || values["enabled"] != true || !equalJSON(t, values["payload"], map[string]any{"items": []any{float64(9)}}) {
		t.Fatal("wrong merged values", values)
	}
	if _, present := values["removed"]; present {
		t.Fatal("null did not delete inherited value")
	}
}
func equalJSON(t *testing.T, a, b any) bool {
	t.Helper()
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}

func TestPresetValuesRefuseUnaddressableBindingNames(t *testing.T) {
	for _, name := range []string{"", "bad.name", "1bad", "bad-name", "$name"} {
		raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "p", "chain": "wbft", "values": map[string]any{name: 1}})
		if _, err := dsl.ParseChainPreset(raw); err == nil || !strings.Contains(err.Error(), "values") {
			t.Errorf("invalid name %q: %v", name, err)
		}
	}
}
