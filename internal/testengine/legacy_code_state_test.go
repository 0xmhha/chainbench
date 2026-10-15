package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// Exercise the public cases through the actual interpreter, with faulted RPC
// observations. The live proof independently hashes the returned runtime code.
func TestLegacyCodeAndStateCasesRejectIncompleteUpgrades(t *testing.T) {
	cases := []string{"02-govminter-v2-code.json", "10-prealloc-preserved-across-boho.json"}
	for _, name := range cases {
		mutations := []string{"valid", "wrong v1 hash", "wrong v2 hash", "empty boundary code", "boundary balance changed"}
		if strings.HasPrefix(name, "10-") {
			mutations = append(mutations, "slot4 changed", "slot12 changed", "slot50 changed", "member slot changed", "getter empty", "getter RPC error")
		}
		for _, mutation := range mutations {
			t.Run(name+"/"+mutation, func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var q struct {
						ID     any    `json:"id"`
						Method string `json:"method"`
						Params []any  `json:"params"`
					}
					if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
						t.Error(err)
						w.WriteHeader(400)
						return
					}
					var result any
					rpcError := false
					switch q.Method {
					case "eth_blockNumber":
						result = "0xb"
					case "eth_chainId":
						result = "0x205b"
					case "net_version":
						result = "8283"
					case "web3_clientVersion":
						result = "gstable/v1.1.0"
					case "eth_getBalance":
						result = "0x20"
						if mutation == "boundary balance changed" && len(q.Params) > 1 && q.Params[1] == "0xa" {
							result = "0x21"
						}
					case "eth_getTransactionCount":
						result = "0x0"
					case "eth_getCode":
						result = "0x6002600055"
						if len(q.Params) > 1 && (q.Params[1] == "0x1" || q.Params[1] == "0x9") {
							result = "0x6001600055"
						}
						if mutation == "empty boundary code" && len(q.Params) > 1 && q.Params[1] == "0xa" {
							result = "0x"
						}
					case "web3_sha3":
						result = "0xf369fcc4edab330169f427fd2aedbe97432cf1cfa31200da06464034f3edaa05"
						if len(q.Params) > 0 && q.Params[0] == "0x" {
							result = "0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
						} else if len(q.Params) > 0 && q.Params[0] == "0x6001600055" {
							result = "0x3ea99dd77ff9bbc4a88b126e9aa06ddbf144f535a5b432040dbedadd66f3c759"
							if mutation == "wrong v1 hash" {
								result = "0x" + strings.Repeat("1", 64)
							}
						} else if mutation == "wrong v2 hash" {
							result = "0x" + strings.Repeat("2", 64)
						}
					case "eth_getStorageAt":
						result = "0x" + fmt.Sprintf("%064x", 2)
						if len(q.Params) > 2 && q.Params[2] == "0xa" {
							slot, _ := q.Params[1].(string)
							slot = strings.TrimLeft(strings.TrimPrefix(slot, "0x"), "0")
							if (mutation == "slot4 changed" && slot == "4") || (mutation == "slot12 changed" && slot == "c") || (mutation == "slot50 changed" && slot == "32") || (mutation == "member slot changed" && len(slot) > 2) {
								result = "0x" + fmt.Sprintf("%064x", 3)
							}
						}
					case "eth_call":
						result = "0x" + strings.Repeat("0", 64)
						if mutation == "getter empty" {
							result = "0x"
						}
						rpcError = mutation == "getter RPC error"
					default:
						rpcError = true
					}
					response := map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result}
					if rpcError {
						delete(response, "result")
						response["error"] = map[string]any{"code": -32000, "message": "explicit fault control"}
					}
					if err := json.NewEncoder(w).Encode(response); err != nil {
						t.Error(err)
					}
				}))
				defer srv.Close()
				raws, err := dsl.ReadFiles([]string{filepath.Join(repoRoot(t), "tests/tc/go-stablenet/post-v1.0.0-change/common-all", name)})
				if err != nil {
					t.Fatal(err)
				}
				engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", RPCURLs: []string{srv.URL}, Caps: []string{"rpc", "fork:boho", "contract:govMinter"}, ArtifactRoot: t.TempDir()})
				if err != nil {
					t.Fatal(err)
				}
				root, err := engine.Run(context.Background(), raws)
				if err != nil {
					t.Fatal(err)
				}
				var session struct {
					Summary struct{ Pass, Fail, Blocked, Skip int }
				}
				b, err := os.ReadFile(filepath.Join(root, "session.json"))
				if err != nil {
					t.Fatal(err)
				}
				if err = json.Unmarshal(b, &session); err != nil {
					t.Fatal(err)
				}
				wantPass, wantFail := 0, 1
				if mutation == "valid" {
					wantPass, wantFail = 1, 0
				}
				if session.Summary.Pass != wantPass || session.Summary.Fail != wantFail || session.Summary.Blocked != 0 || session.Summary.Skip != 0 {
					t.Fatalf("fault %q hidden by public case: %+v", mutation, session.Summary)
				}
			})
		}
	}
}
