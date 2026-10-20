package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestKeptOutRequiresAuthoritativeReceiptObservation(t *testing.T) {
	for _, mode := range []string{"unknown RPC", "unknown HTTP", "known absent", "known mined"} {
		t.Run(mode, func(t *testing.T) {
			var heads atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ID     any    `json:"id"`
					Method string `json:"method"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				response := map[string]any{"jsonrpc": "2.0", "id": request.ID}
				switch request.Method {
				case "eth_blockNumber":
					response["result"] = fmt.Sprintf("0x%x", heads.Add(1))
				case "eth_getTransactionReceipt":
					switch mode {
					case "unknown RPC":
						response["error"] = map[string]any{"code": -32000, "message": "receipt backend unavailable"}
					case "unknown HTTP":
						w.WriteHeader(http.StatusServiceUnavailable)
						response["result"] = nil
					case "known absent":
						response["result"] = nil
					case "known mined":
						response["result"] = map[string]any{"transactionHash": "0xqueued", "status": "0x1"}
					}
				default:
					t.Errorf("unexpected RPC %s", request.Method)
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			ac := &interp.ActionCtx{Args: map[string]any{"blocks": 1, "timeout": "100ms", "pollInterval": "1ms"}}
			err := checkKeptOut(context.Background(), rpc.Dial(server.URL), "0xqueued", nil, ac)
			if (err == nil) != (mode == "known absent") {
				t.Fatalf("receipt %s: outcome=%v", mode, err)
			}
			if mode != "known absent" && ac.Value != nil {
				t.Fatalf("unknown/mined receipt bound success evidence: %v", ac.Value)
			}
		})
	}
}
