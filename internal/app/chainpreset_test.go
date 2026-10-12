package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestChainPresetsRegisteredValidOnly(t *testing.T) {
	root := t.TempDir()
	for name, raw := range map[string]string{
		"valid.json":       `{"schemaVersion":"2","kind":"chain-preset","id":"valid","chain":"wbft","topology":{"bp":4}}`,
		"unsupported.json": `{"schemaVersion":"2","kind":"chain-preset","id":"unsupported","chain":"typo"}`,
		"invalid.json":     `{"schemaVersion":"2","kind":"chain-preset","id":"invalid","chain":"wbft","topology":{"bp":0}}`,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	}
	presets, err := ChainPresets(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(presets) != 1 || presets[0].ID != "valid" {
		t.Fatalf("presets: %v", presets)
	}
	if err := ValidateChainPreset(presets[0].Content); err != nil {
		t.Fatal(err)
	}
	if _, err := ChainPresets(filepath.Join(root, "missing")); err == nil {
		t.Fatal("hidden I/O failure")
	}
}

func TestChainPresetContractVocabulary(t *testing.T) {
	for _, chain := range []string{"stablenet", "wbft", "wemix"} {
		schema, err := ChainPresetContract(chain)
		if err != nil {
			t.Fatal(err)
		}
		props := schema["$defs"].(map[string]any)["envSpec"].(map[string]any)["properties"].(map[string]any)
		launch := props["launch"].(map[string]any)
		all := launch["properties"].(map[string]any)["all"].(map[string]any)["properties"].(map[string]any)
		single := launch["additionalProperties"].(map[string]any)["properties"].(map[string]any)
		if len(single) != map[string]int{"stablenet": 78, "wbft": 78, "wemix": 87}[chain] {
			t.Fatalf("incomplete vocabulary for %s: %d", chain, len(single))
		}
		if _, ok := all["http.port"]; ok {
			t.Fatal("per-node port offered to all nodes")
		}
		if _, ok := single["http.port"]; !ok {
			t.Fatal("missing per-node port")
		}
		_, ext := single["chain.consensusmethod"]
		if ext != (chain == "wemix") {
			t.Fatal("dialect leaked")
		}
		if _, ok := single["docroot"]; ok {
			t.Fatal("unmapped binary option exposed")
		}
	}
	if _, err := ChainPresetContract("unknown"); err == nil {
		t.Fatal("unsupported chain accepted")
	}
}
