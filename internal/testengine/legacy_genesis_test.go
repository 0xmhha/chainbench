package testengine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestLegacyGenesisRejectsIncompleteNodeAgreement runs the complete public case
// including its node targeting. Equality of the available hashes must not
// hide a node that failed to return its genesis block.
func TestLegacyGenesisRejectsIncompleteNodeAgreement(t *testing.T) {
	for _, fault := range []string{"valid", "secondary empty hash", "primary empty hash", "secondary null block", "all short hashes", "secondary wrong parent", "primary wrong parent", "secondary different hash", "secondary wrong chain", "primary bare hash"} {
		t.Run(fault, func(t *testing.T) {
			urls := []string{}
			for node := range 4 {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var q struct {
						ID     any
						Method string
						Params []any
					}
					if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
						t.Error(err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					var result any
					switch q.Method {
					case "eth_chainId":
						result = "0x205b"
						if fault == "secondary wrong chain" && node == 2 {
							result = "0x205c"
						}
					case "eth_blockNumber":
						result = "0x10"
					case "net_version":
						result = "8283"
					case "web3_clientVersion":
						result = "gstable/v1.1.0"
					case "eth_getBlockByNumber":
						block := map[string]any{"number": "0x0", "hash": "0x" + strings.Repeat("ab", 32), "parentHash": "0x" + strings.Repeat("0", 64)}
						if len(q.Params) != 2 || q.Params[0] != "0x0" {
							block["number"] = "0x10"
						} else {
							switch {
							case fault == "secondary null block" && node == 2:
								block = nil
							case fault == "secondary empty hash" && node == 2, fault == "primary empty hash" && node == 0:
								block["hash"] = ""
							case fault == "all short hashes":
								block["hash"] = "0xdead"
							case fault == "secondary wrong parent" && node == 2, fault == "primary wrong parent" && node == 0:
								block["parentHash"] = "0x" + strings.Repeat("1", 64)
							case fault == "secondary different hash" && node == 2:
								block["hash"] = "0x" + strings.Repeat("cd", 32)
							case fault == "primary bare hash" && node == 0:
								block["hash"] = "0x"
							}
						}
						result = block
					default:
						_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "error": map[string]any{"code": -32601, "message": "unexpected mock RPC"}})
						return
					}
					if err := json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result}); err != nil {
						t.Error(err)
					}
				}))
				t.Cleanup(srv.Close)
				urls = append(urls, srv.URL)
			}
			raws, err := dsl.ReadFiles([]string{filepath.Join(repoRoot(t), "tests/tc/common/node/CT-NODE-003-genesis-init.json")})
			if err != nil {
				t.Fatal(err)
			}
			engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", RPCURLs: urls, Caps: []string{"rpc"}, ArtifactRoot: t.TempDir()})
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
			if fault == "valid" {
				wantPass, wantFail = 1, 0
			}
			if session.Summary.Pass != wantPass || session.Summary.Fail != wantFail || session.Summary.Blocked != 0 || session.Summary.Skip != 0 {
				t.Fatalf("public genesis case hid %q: %+v", fault, session.Summary)
			}
		})
	}
}
