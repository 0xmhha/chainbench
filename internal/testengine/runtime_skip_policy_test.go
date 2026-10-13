package testengine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestDeclaredRuntimeSkipIsJudgedAfterActualCapabilityProbe(t *testing.T) {
	valid, _ := p256CorpusInputs(t)
	for _, capability := range []string{"consensus", "evm:shanghai", "precompile:secp256r1"} {
		for _, supported := range []bool{false, true} {
			name := capability + "-absent"
			if supported {
				name = capability + "-present"
			}
			t.Run(name, func(t *testing.T) {
				var probes, steps atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var request struct {
						ID     any
						Method string
						Params []json.RawMessage
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						return
					}
					response := map[string]any{"jsonrpc": "2.0", "id": request.ID}
					switch request.Method {
					case "istanbul_getValidators":
						probes.Add(1)
						if supported {
							response["result"] = []string{"0xpublic-validator"}
						} else {
							response["error"] = map[string]any{"code": -32601, "message": "method not found"}
						}
					case "eth_call":
						probes.Add(1)
						if capability == "evm:shanghai" {
							if supported {
								response["result"] = "0x" + strings.Repeat("0", 64)
							} else {
								response["error"] = map[string]any{"code": -32000, "message": "invalid opcode: PUSH0"}
							}
						} else {
							var call map[string]any
							_ = json.Unmarshal(request.Params[0], &call)
							response["result"] = "0x"
							if supported && call["data"] == valid {
								response["result"] = "0x" + strings.Repeat("0", 63) + "1"
							}
						}
					default:
						steps.Add(1)
						response["result"] = "0x10"
					}
					_ = json.NewEncoder(w).Encode(response)
				}))
				defer server.Close()
				eng, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "wbft", RPCURLs: []string{server.URL}, Caps: []string{capability}, ArtifactRoot: t.TempDir()})
				if err != nil {
					t.Fatal(err)
				}
				raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "runtime-skip", "requires": []string{"rpc", capability}, "skipsOn": []string{"wbft"}, "chainPreset": map[string]any{"chain": "wbft"}, "hooks": map[string]any{"pre": []any{map[string]any{"do": "read", "source": "blockNumber", "save": "height"}}}, "steps": []any{map[string]any{"expect": "blockNumber", "is": 16}}})
				root, err := eng.Run(context.Background(), [][]byte{raw})
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(filepath.Join(root, "tests", "001_runtime-skip", "status.json"))
				if err != nil {
					t.Fatal(err)
				}
				var verdict struct{ Result, Reason string }
				_ = json.Unmarshal(data, &verdict)
				want := "skip"
				if supported {
					want = "fail"
				}
				if verdict.Result != want || probes.Load() == 0 || steps.Load() != 0 {
					t.Fatalf("skip judged before actual support: %s probes=%d steps=%d", data, probes.Load(), steps.Load())
				}
				if supported && !strings.Contains(verdict.Reason, "remove this chain from skipsOn") {
					t.Fatalf("missing stale declaration reason: %s", data)
				}
			})
		}
	}
}
