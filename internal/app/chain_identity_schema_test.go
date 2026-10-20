package app

import "testing"

func TestEditorChainIdentityMapAppliesItsPatternContract(t *testing.T) {
	schema := map[string]any{"type": "object", "patternProperties": map[string]any{"^chainId$": map[string]any{"type": "integer", "minimum": 0}}, "additionalProperties": true}
	if err := validateEditorSchema([]byte(`{"chainId":"unusable"}`), schema, schema); err == nil {
		t.Fatal("declared identity pattern was ignored")
	}
	if err := validateEditorSchema([]byte(`{"chainId":0,"other":"arbitrary"}`), schema, schema); err != nil {
		t.Fatal(err)
	}
}
