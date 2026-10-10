package testengine_test

import (
	"encoding/json"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestValidateChainPreset(t *testing.T) {
	for _, chain := range []string{"stablenet", "wbft", "wemix"} {
		t.Run(chain, func(t *testing.T) {
			env := map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "test", "chain": chain, "topology": map[string]any{"bp": 4, "en": 1, "syncMode": "snap"}}
			raw, _ := json.Marshal(env)
			if err := testengine.ValidateChainPreset(raw); err != nil {
				t.Fatal(err)
			}
			env["launch"] = map[string]any{"all": map[string]any{"chain.block.interval": "1"}}
			raw, _ = json.Marshal(env)
			err := testengine.ValidateChainPreset(raw)
			if (err == nil) != (chain == "wemix") {
				t.Fatalf("dialect-specific override: %v", err)
			}
		})
	}
}

func TestValidateChainPresetRefusesUnsupportedDeclarations(t *testing.T) {
	base := `{"schemaVersion":"2","kind":"chain-preset","id":"test","chain":"stablenet"`
	for _, tail := range []string{
		`,"unknown":true}`, `,"topology":{"bp":0}}`, `,"topology":{"bp":1.5}}`,
		`,"topology":{"bp":1,"syncMode":"typo"}}`, `,"topology":{"invalid":2}}`,
		`,"launch":{"all":{"docroot":"x"}}}`, `,"launch":{"all":{"port":30301}}}`,
		`,"launch":{"all":{"metrics.port":6060}}}`, `,"launch":{"all":{"http":"true"}}}`,
		`,"launch":{"all":{"http":false}}}`, `,"launch":{"all":{"cache":[]}}}`,
		`,"launch":{"node99":{"cache":1}}}`,
		`,"launch":{"invalid":{"cache":1}}}`, `,"config":{"all":{"unknown":"x"}}}`,
	} {
		t.Run(tail, func(t *testing.T) {
			if err := testengine.ValidateChainPreset([]byte(base + tail)); err == nil {
				t.Fatal("accepted unsupported declaration")
			}
		})
	}
	for _, raw := range []string{
		`{"schemaVersion":"1","kind":"chain-preset","id":"x","chain":"stablenet"}`,
		`{"schemaVersion":"2","kind":"chain-preset","id":"x","chain":"unsupported"}`,
		base + `} {}`, base + `,"topology":{"nodes":[]}}`,
	} {
		if err := testengine.ValidateChainPreset([]byte(raw)); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestValidateChainPresetLargeCountDoesNotAllocateNodes(t *testing.T) {
	raw := []byte(`{"schemaVersion":"2","kind":"chain-preset","id":"large","chain":"wbft","topology":{"bp":100000000},"launch":{"node100000000":{"cache":64}}}`)
	if err := testengine.ValidateChainPreset(raw); err != nil {
		t.Fatal(err)
	}
}
