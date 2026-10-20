package testhelper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestPublicBlockLookupRejectsInconsistentFullBlocks(t *testing.T) {
	for _, name := range []string{"valid", "empty extra", "different extra", "different miner", "missing miner", "different hash", "different number", "different transactions", "hash only transactions", "empty transactions"} {
		t.Run(name, func(t *testing.T) {
			hash := "0x" + strings.Repeat("ab", 32)
			txhash := "0x" + strings.Repeat("cd", 32)
			protocol := []string{}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct {
					ID     any
					Method string
					Params []any
				}
				if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
					t.Error(err)
					return
				}
				var result any
				switch q.Method {
				case "eth_getTransactionReceipt":
					result = map[string]any{"blockNumber": "0xa", "status": "0x1"}
				case "eth_blockNumber":
					result = "0xa"
				default:
					block := map[string]any{"number": "0xa", "hash": hash, "extraData": "0x1234", "miner": "0x" + strings.Repeat("12", 20), "transactions": []any{map[string]any{"hash": txhash, "blockHash": hash, "blockNumber": "0xa", "transactionIndex": "0x0"}}}
					if len(q.Params) != 2 || q.Params[1] != true {
						protocol = append(protocol, "fullTransactions must be true")
					}
					if q.Method == "eth_getBlockByNumber" && q.Params[0] != "0xa" {
						protocol = append(protocol, "number must be pinned")
					}
					if name == "empty extra" {
						block["extraData"] = "0x"
					}
					if q.Method == "eth_getBlockByHash" {
						switch name {
						case "different extra":
							block["extraData"] = "0x5678"
						case "different miner":
							block["miner"] = "0x" + strings.Repeat("34", 20)
						case "missing miner":
							delete(block, "miner")
						case "different hash":
							block["hash"] = "0x" + strings.Repeat("ef", 32)
						case "different number":
							block["number"] = "0xb"
						case "different transactions":
							block["transactions"] = []any{map[string]any{"hash": "other"}}
						}
					}
					if name == "hash only transactions" {
						block["transactions"] = []any{txhash}
					}
					if name == "empty transactions" {
						block["transactions"] = []any{}
					}
					result = block
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
			}))
			defer srv.Close()
			raws, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/rpc/CT-RPC-003-block-by-hash-consistency.json"}, "../../presets/chain/stablenet-bp4.json")
			if err != nil {
				t.Fatal(err)
			}
			s, err := dsl.ParseV2(raws[0])
			if err != nil {
				t.Fatal(err)
			}
			selected := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Do == "newAccount" || st.Do == "sendTx" {
					continue
				}
				selected = append(selected, st)
			}
			s.Sequence = selected
			s.EnvValues = map[string]any{"probeHash": txhash}
			rec := &recordStub{}
			got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			if name == "valid" {
				if err != nil || got != session.StatusPass || len(protocol) != 0 {
					t.Fatalf("valid block rejected or wrong protocol: status=%s err=%v protocol=%v assertions=%+v", got, err, protocol, rec.asserts)
				}
			} else if err == nil && got == session.StatusPass {
				t.Fatalf("invalid %s accepted: %+v", name, rec.asserts)
			}
		})
	}
}
