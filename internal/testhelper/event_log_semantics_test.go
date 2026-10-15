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

func TestPublicEventLogsBindReturnedFieldsToReceipt(t *testing.T) {
	for _, name := range []string{"valid", "wrong address", "wrong height", "wrong block hash", "wrong transaction hash", "removed", "wrong data"} {
		t.Run(name, func(t *testing.T) {
			addr := "0x" + strings.Repeat("12", 20)
			tx := "0x" + strings.Repeat("ab", 32)
			block := "0x" + strings.Repeat("cd", 32)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct {
					ID     any
					Method string
					Params []any
				}
				if e := json.NewDecoder(r.Body).Decode(&q); e != nil {
					t.Error(e)
					return
				}
				var result any
				switch q.Method {
				case "eth_getTransactionReceipt":
					result = map[string]any{"contractAddress": addr, "status": "0x1", "blockNumber": "0xa", "blockHash": block, "transactionHash": tx}
				case "eth_getLogs":
					filter := q.Params[0].(map[string]any)
					if filter["address"] != addr || filter["fromBlock"] != "0xa" || filter["toBlock"] != "0xa" {
						t.Errorf("wrong filter: %v", filter)
					}
					log := map[string]any{"address": addr, "blockNumber": "0xa", "blockHash": block, "transactionHash": tx, "transactionIndex": "0x0", "logIndex": "0x0", "removed": false, "data": "0x", "topics": []any{"0x" + strings.Repeat("11", 32)}}
					switch name {
					case "wrong address":
						log["address"] = "0x" + strings.Repeat("34", 20)
					case "wrong height":
						log["blockNumber"] = "0xb"
					case "wrong block hash":
						log["blockHash"] = "0x" + strings.Repeat("ef", 32)
					case "wrong transaction hash":
						log["transactionHash"] = "0x" + strings.Repeat("ef", 32)
					case "removed":
						log["removed"] = true
					case "wrong data":
						log["data"] = "0x01"
					}
					result = []any{log}
					if topics, ok := filter["topics"].([]any); ok && len(topics) > 0 && topics[0] == "0x"+strings.Repeat("22", 32) {
						result = []any{}
					}
				default:
					t.Errorf("unexpected RPC %s", q.Method)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
			}))
			defer srv.Close()
			raws, e := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/rpc/CT-RPC-009-event-logs-query.json"}, "../../presets/chain/stablenet-bp4.json")
			if e != nil {
				t.Fatal(e)
			}
			s, e := dsl.ParseV2(raws[0])
			if e != nil {
				t.Fatal(e)
			}
			selected := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Do != "sendTx" {
					selected = append(selected, st)
				}
			}
			s.Sequence = selected
			s.EnvValues = map[string]any{"dhash": tx, "ehash": tx}
			rec := &recordStub{}
			got, e := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			if name == "valid" {
				if e != nil || got != session.StatusPass {
					t.Fatalf("valid log failed: %s %v %+v", got, e, rec.asserts)
				}
			} else if e == nil && got == session.StatusPass {
				t.Fatalf("invalid returned %s passed: %+v", name, rec.asserts)
			}
		})
	}
}
