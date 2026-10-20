package app

import (
	"encoding/json"
	"testing"
)

func TestPresetNamesDoNotChangeExecutableSemantics(t *testing.T) {
	var doc map[string]any
	_ = json.Unmarshal(editorCase(`[{"expect":"blockNumber","is":0}]`), &doc)
	fingerprints := []string{}
	for _, id := range []string{"first", "second"} {
		doc["chainPreset"].(map[string]any)["id"] = id
		raw, _ := json.Marshal(doc)
		prepared, err := PrepareTestCase(TestCaseInput{Content: raw})
		if err != nil {
			t.Fatal(err)
		}
		fingerprints = append(fingerprints, prepared.SemanticFingerprint)
	}
	if fingerprints[0] != fingerprints[1] {
		t.Fatal("preset provenance changed execution semantics", fingerprints)
	}
}
