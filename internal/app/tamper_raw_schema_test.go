package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/testhelper"
)

func TestTamperRawEditorContractKeepsSigningModesExclusive(t *testing.T) {
	_, schemas, err := testhelper.EditorVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	schema := schemas["action-sendRawTampered"]
	for _, c := range []struct {
		input string
		valid bool
	}{
		{`{"do":"sendRawTampered","raw":"$nodeSigned","which":"sender"}`, true},
		{`{"do":"sendRawTampered","raw":"${nodeSigned}","which":"feepayer"}`, true},
		{`{"do":"sendRawTampered","raw":"0x16c0","which":"sender"}`, true},
		{`{"do":"sendRawTampered","senderKey":"$s","feePayerKey":"$p","to":"node1","which":"sender"}`, true},
		{`{"do":"sendRawTampered","raw":"$nodeSigned","senderKey":"$s","which":"sender"}`, false},
		{`{"do":"sendRawTampered","raw":"$nodeSigned","feePayerKey":"$p","which":"sender"}`, false},
		{`{"do":"sendRawTampered","raw":"$nodeSigned","to":"node1","which":"sender"}`, false},
		{`{"do":"sendRawTampered","raw":"$nodeSigned","value":"1","which":"sender"}`, false},
		{`{"do":"sendRawTampered","raw":"","which":"sender"}`, false},
		{`{"do":"sendRawTampered","raw":null,"which":"sender"}`, false},
		{`{"do":"sendRawTampered","raw":"0x16zz","which":"sender"}`, false},
		{`{"do":"sendRawTampered","raw":"$nodeSigned","which":"other"}`, false},
		{`{"do":"sendRawTampered","raw":"$nodeSigned"}`, false},
		{`{"do":"sendRawTampered","which":"sender"}`, false},
		{`{"do":"sendRawTampered","senderKey":"","feePayerKey":"$p","to":"node1","which":"sender"}`, false},
	} {
		if err := validateEditorSchema(json.RawMessage(c.input), schema, schema); (err == nil) != c.valid {
			t.Errorf("valid=%v want %v: %s: %v", err == nil, c.valid, c.input, err)
		}
	}
}

func TestPersonalDelegationCasesRoundTripThroughEditorContract(t *testing.T) {
	presets := map[string]json.RawMessage{}
	paths, err := filepath.Glob("../../presets/chain/*.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		presets[strings.TrimSuffix(filepath.Base(path), ".json")] = b
	}
	for _, name := range []string{"01b-personal-normal.json", "02b-personal-sender.json", "03b-personal-feepayer.json", "04b-personal-insufficient.json"} {
		t.Run(name, func(t *testing.T) {
			b, err := os.ReadFile(filepath.Join("../../tests/tc/go-stablenet/regression/fee-delegation", name))
			if err != nil {
				t.Fatal(err)
			}
			prepared, err := PrepareTestCase(TestCaseInput{Content: b, Presets: presets})
			if err != nil {
				t.Fatal(err)
			}
			again, err := PrepareTestCase(TestCaseInput{Content: prepared.Content, Presets: presets})
			if err != nil || again.SemanticFingerprint != prepared.SemanticFingerprint {
				t.Fatal("editor round trip changed executable semantics", err)
			}
		})
	}
}
