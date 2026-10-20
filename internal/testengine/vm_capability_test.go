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

func TestShanghaiDeclarationRequiresActualOpcodeExecution(t *testing.T) {
	for _, c := range []struct {
		name, reply, status string
		code                int
	}{
		{"supported", "supported", "pass", -32000},
		{"unsupported", "invalid opcode: PUSH0", "skip", -32000},
		{"hex opcode", "invalid opcode: opcode 0x5f not defined", "skip", -32000},
		{"other opcode", "invalid opcode: opcode 0xaa not defined", "blocked", -32000},
		{"invalid params", "invalid opcode: PUSH0", "blocked", -32602},
		{"missing probe API", "method not found", "blocked", -32601},
		{"unrelated execution failure", "insufficient funds", "blocked", -32000},
		{"unusable result", "empty", "blocked", -32000},
		{"transport failure", "http", "blocked", -32000},
	} {
		t.Run(c.name, func(t *testing.T) {
			var probes, steps atomic.Int32
			urls := []string{}
			for i := 0; i < 2; i++ {
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
					if request.Method != "eth_call" {
						if request.Method == "test_hook" || request.Method == "test_body" {
							steps.Add(1)
						}
						_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": "0x10"})
						return
					}
					probes.Add(1)
					var call map[string]any
					_ = json.Unmarshal(request.Params[0], &call)
					if call["data"] != "0x5f60005260206000f3" || call["gas"] != "0x186a0" || len(request.Params) != 2 || string(request.Params[1]) != `"latest"` {
						t.Errorf("probe did not execute the bounded opcode: %+v", request)
					}
					if i == 1 && c.reply == "http" {
						http.Error(w, "private details", http.StatusServiceUnavailable)
						return
					}
					if i == 1 && c.reply != "supported" && c.reply != "empty" {
						_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "error": map[string]any{"code": c.code, "message": c.reply}})
						return
					}
					result := "0x" + strings.Repeat("0", 64)
					if i == 1 && c.reply == "empty" {
						result = "0x"
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
				}))
				defer server.Close()
				urls = append(urls, server.URL)
			}
			eng, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", RPCURLs: urls, Caps: []string{"evm:shanghai"}, ArtifactRoot: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			raw := []byte(`{"schemaVersion":"2","kind":"case","id":"vm-probe","requires":["rpc","evm:shanghai"],"chainPreset":{"chain":"stablenet"},"hooks":{"pre":[{"do":"read","source":"rpcCall","method":"test_hook","save":"height"}]},"steps":[{"expect":"rpcCall","method":"test_body","on":"node2","is":"0x10"}]}`)
			root, err := eng.Run(context.Background(), [][]byte{raw})
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "tests", "001_vm-probe")
			data, err := os.ReadFile(filepath.Join(dir, "status.json"))
			if err != nil {
				t.Fatal(err)
			}
			var verdict struct{ Result string }
			_ = json.Unmarshal(data, &verdict)
			if verdict.Result != c.status || probes.Load() != 2 {
				t.Errorf("declaration hid actual VM support: %s probes=%d", data, probes.Load())
			}
			if c.status != "pass" && steps.Load() != 0 {
				t.Error("unqualified VM executed the case")
			}
			data, err = os.ReadFile(filepath.Join(dir, "observations", "vm-capability-probes.json"))
			if err != nil {
				t.Fatal(err)
			}
			var observation struct {
				Capability string
				Nodes      []struct {
					Node           int
					Source, Status string
				}
			}
			if err := json.Unmarshal(data, &observation); err != nil {
				t.Fatal(err)
			}
			if observation.Capability != "evm:shanghai" || len(observation.Nodes) != 2 || observation.Nodes[0].Status != "supported" {
				t.Fatalf("missing opcode provenance: %s", data)
			}
		})
	}
}
