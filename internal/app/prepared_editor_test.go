package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPreparedConsumerCanBeAuthoredBeforeItsExecutionListIsSelected(t *testing.T) {
	input := TestCaseInput{Content: json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"consume","chainPreset":{"chain":"wbft"},"prepared":{"imports":{"head":{"case":"prepare","name":"head"}}},"steps":[{"expect":"blockNumber","is":"$head"}]}`)}
	first, err := PrepareWebTestCase(input)
	if err != nil {
		t.Fatal("consumer declaration wrongly needs its runtime producer", err)
	}
	if string(first.Content) != string(input.Content) {
		t.Fatal("editor changed the explicit preparation reference")
	}
	input.Content = json.RawMessage(strings.Replace(string(input.Content), `"case":"prepare"`, `"case":"other"`, 1))
	second, err := PrepareWebTestCase(input)
	if err != nil {
		t.Fatal(err)
	}
	if first.SemanticFingerprint == second.SemanticFingerprint {
		t.Fatal("changing the preparation source did not change case semantics")
	}
}
