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

func TestPublicReceiptBindsActualBlockAndPositiveGas(t *testing.T) {
	for _, name := range []string{"valid", "wrong receipt block hash", "wrong receipt number", "missing receipt block hash", "zero gas", "invalid gas", "wrong transaction block hash", "wrong transaction block number"} {
		t.Run(name, func(t *testing.T) {
			block := "0x" + strings.Repeat("ab", 32)
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
					receipt := map[string]any{"status": "0x1", "effectiveGasPrice": "0x64", "logs": []any{}, "blockNumber": "0x5", "blockHash": block, "gasUsed": "0x5208"}
					switch name {
					case "wrong receipt block hash":
						receipt["blockHash"] = "0x" + strings.Repeat("cd", 32)
					case "wrong receipt number":
						receipt["blockNumber"] = "0x6"
					case "missing receipt block hash":
						delete(receipt, "blockHash")
					case "zero gas":
						receipt["gasUsed"] = "0x0"
					case "invalid gas":
						receipt["gasUsed"] = "bad"
					}
					result = receipt
				case "eth_getBlockByNumber":
					result = map[string]any{"number": "0x5", "hash": block}
				case "eth_getTransactionByHash":
					tx := map[string]any{"blockNumber": "0x5", "blockHash": block}
					if name == "wrong transaction block hash" {
						tx["blockHash"] = "0x" + strings.Repeat("cd", 32)
					}
					if name == "wrong transaction block number" {
						tx["blockNumber"] = "0x6"
					}
					result = tx
				default:
					t.Error("unexpected receipt probe", q.Method)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
			}))
			defer srv.Close()
			raws, e := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/rpc/CT-RPC-005-transaction-receipt-fields.json"}, "../../presets/chain/stablenet-bp4.json")
			if e != nil {
				t.Fatal(e)
			}
			s, e := dsl.ParseV2(raws[0])
			if e != nil {
				t.Fatal(e)
			}
			sequence := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Do != "newAccount" && st.Do != "sendTx" {
					sequence = append(sequence, st)
				}
			}
			s.Sequence = sequence
			s.EnvValues = map[string]any{"hash": "tx"}
			rec := &recordStub{}
			got, e := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			if name == "valid" {
				if e != nil || got != session.StatusPass {
					t.Fatalf("valid receipt failed: %s %v %+v", got, e, rec.asserts)
				}
			} else if e == nil && got == session.StatusPass {
				t.Fatalf("invalid %s accepted: %+v", name, rec.asserts)
			}
		})
	}
}
