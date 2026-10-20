package testengine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestAttachCapabilitiesUseActualRPCInsteadOfDeclaredClaims(t *testing.T) {
	for _, scenario := range []struct {
		name             string
		transportFailure bool
		supported        bool
		want             string
	}{{"method missing", false, false, "skip"}, {"probe unavailable", true, false, "blocked"}, {"method supported", false, true, "pass"}} {
		t.Run(scenario.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ID     any    `json:"id"`
					Method string `json:"method"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				if request.Method == "istanbul_getValidators" {
					if scenario.supported {
						_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": []string{"0xpublic-validator"}})
						return
					}
					if scenario.transportFailure {
						http.Error(w, "temporarily unavailable", http.StatusServiceUnavailable)
						return
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "error": map[string]any{"code": -32601, "message": "method not found"}})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": "0x10"})
			}))
			defer server.Close()
			engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", RPCURLs: []string{server.URL}, Caps: []string{"consensus"}, ArtifactRoot: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			raw := []byte(`{"schemaVersion":"2","kind":"case","id":"actual-capability","requires":["rpc","consensus"],"chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"external","chain":"stablenet","attach":{"rpc":["http://unused"]}},"steps":[{"expect":"blockNumber","compare":"GreaterOrEqual","is":"1"}]}`)
			root, err := engine.Run(context.Background(), [][]byte{raw})
			if err != nil {
				t.Fatal(err)
			}
			bytes, err := os.ReadFile(filepath.Join(root, "tests", "001_actual-capability", "status.json"))
			if err != nil {
				t.Fatal(err)
			}
			var status struct {
				Result string `json:"result"`
				Reason string `json:"reason"`
			}
			if err := json.Unmarshal(bytes, &status); err != nil {
				t.Fatal(err)
			}
			if status.Result != scenario.want || (scenario.want != "pass" && status.Reason == "") {
				t.Fatalf("declared claim hid actual RPC capability: %s", bytes)
			}
		})
	}
}
