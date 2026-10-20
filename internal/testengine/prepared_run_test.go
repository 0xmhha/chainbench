package testengine_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestPreparedValuesReachAnotherCaseAndStayInTheirSession(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "published", true: "producer failed"}[failed], func(t *testing.T) {
			var consumed atomic.Int32
			address := "0x000000000000000000000000000000000000dead"
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     any
					Method string
					Params []any
				}
				_ = json.NewDecoder(r.Body).Decode(&req)
				var result any = "0x1"
				switch req.Method {
				case "fixture_prepare":
					result = address
				case "fixture_consume":
					consumed.Add(1)
					if len(req.Params) != 1 || req.Params[0] != address {
						t.Error("consumer did not use the producer's typed value", req.Params)
					}
					result = address
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
			}))
			defer server.Close()
			engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "wbft", RPCURLs: []string{server.URL}, ArtifactRoot: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			caseRaw := func(id string, prepared map[string]any, steps []any) []byte {
				raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": id, "requires": []string{"rpc"}, "chainPreset": map[string]any{"chain": "wbft"}, "prepared": prepared, "steps": steps})
				if err != nil {
					t.Fatal(err)
				}
				return raw
			}
			want := address
			if failed {
				want = "wrong address"
			}
			producer := caseRaw("prepare", map[string]any{"exports": map[string]any{"contract": "$deployed"}}, []any{map[string]any{"do": "read", "source": "rpcCall", "method": "fixture_prepare", "save": "deployed"}, map[string]any{"expect": "rpcCall", "method": "fixture_prepare", "is": want}})
			consumer := caseRaw("consume", map[string]any{"imports": map[string]any{"address": map[string]any{"case": "prepare", "name": "contract"}}}, []any{map[string]any{"expect": "rpcCall", "method": "fixture_consume", "params": []any{"$address"}, "is": address}})
			root, err := engine.Run(t.Context(), [][]byte{producer, consumer})
			if err != nil {
				t.Fatal(err)
			}
			result := func(root, name string) string {
				data, err := os.ReadFile(filepath.Join(root, "tests", name, "status.json"))
				if err != nil {
					t.Fatal(err)
				}
				var status struct{ Result string }
				if err := json.Unmarshal(data, &status); err != nil {
					t.Fatal(err)
				}
				return status.Result
			}
			if failed {
				if result(root, "001_prepare") != "fail" || result(root, "002_consume") != "blocked" || consumed.Load() != 0 {
					t.Fatal("failed preparation was shared or consumer ran")
				}
			} else {
				if result(root, "001_prepare") != "pass" || result(root, "002_consume") != "pass" || consumed.Load() != 1 {
					t.Fatal("successful preparation did not reach its consumer")
				}
			}
			before := consumed.Load()
			second, err := engine.Run(t.Context(), [][]byte{consumer})
			if err != nil {
				t.Fatal(err)
			}
			if result(second, "001_consume") != "blocked" || consumed.Load() != before {
				t.Fatal("preparation leaked into a later session")
			}
		})
	}
}
