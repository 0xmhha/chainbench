package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEditorPresetNullValuesMustActuallyBeNull(t *testing.T) {
	schema := map[string]any{"type": "null"}
	for _, value := range []any{false, 1, "", map[string]any{}, []any{}} {
		raw, _ := json.Marshal(value)
		if err := validateEditorSchema(raw, schema, schema); err == nil {
			t.Errorf("non-null accepted: %T", value)
		}
	}
	if err := validateEditorSchema([]byte("null"), schema, schema); err != nil {
		t.Fatal(err)
	}
}

func TestWebInitialValuesCannotPublishSigningKeyMaterial(t *testing.T) {
	key := "0x" + strings.Repeat("ab", 32)
	preset := json.RawMessage(`{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"wbft","values":{"secret":"` + key + `"}}`)
	for _, field := range []string{"key", "feePayerKey", "senderKey", "authorityKey"} {
		content := map[string]any{"chainPreset": map[string]any{"chain": "wbft", "values": map[string]any{"secret": key}}, "steps": []any{map[string]any{"do": "sendTx", field: "$secret"}}}
		if err := validateWebDocumentSecrets("case", content); err == nil || strings.Contains(err.Error(), key) {
			t.Errorf("initial key material publishable or echoed (%s): %v", field, err)
		}
	}
	input := TestCaseInput{Content: json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"c","chainPreset":"p","steps":[{"do":"sendTx","key":"$secret","to":"0x000000000000000000000000000000000000dead","value":"1"},{"expect":"blockNumber","is":0}]}`), Presets: map[string]json.RawMessage{"p": preset}}
	if prepared, err := PrepareWebTestCase(input); err == nil || len(prepared.Content) != 0 || strings.Contains(err.Error(), key) {
		t.Fatal("inherited initial key material publishable", err)
	}
	// The same public hex is valid as a hash or storage key, without signing use.
	input.Content = json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"c","chainPreset":"p","steps":[{"expect":"blockNumber","is":0}]}`)
	if _, err := PrepareWebTestCase(input); err != nil {
		t.Fatal("public hash classified as secret", err)
	}
}

func TestPresetValuesChangeCaseSemanticsAndRetainTheDeclaration(t *testing.T) {
	input := TestCaseInput{Content: json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"c","chainPreset":{"chain":"wbft","values":{"target":2}},"steps":[{"expect":"blockNumber","is":"$target"}]}`)}
	first, err := PrepareWebTestCase(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(first.Content) != string(input.Content) {
		t.Fatal("initial declaration changed")
	}
	input.Content = json.RawMessage(strings.Replace(string(input.Content), `"target":2`, `"target":3`, 1))
	second, err := PrepareWebTestCase(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.SemanticFingerprint == second.SemanticFingerprint {
		t.Fatal("changed initial value retained the old execution fingerprint")
	}
}
