package testhelper

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestLocalAssetEstimationRetainsCreationIntentAndPropagatesFailure(t *testing.T) {
	for _, refused := range []bool{false, true} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var request struct {
				ID     any              `json:"id"`
				Method string           `json:"method"`
				Params []map[string]any `json:"params"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
				return
			}
			if request.Method != "eth_estimateGas" || len(request.Params) != 1 {
				t.Errorf("unexpected RPC: %+v", request)
				return
			}
			args := request.Params[0]
			if args["from"] != testTo || args["data"] != "0x6000" || args["value"] != "0x5" || args["nonce"] != "0x7" || args["gasPrice"] != "0x3" {
				t.Errorf("lost estimate fields: %+v", args)
			}
			if _, has := args["to"]; has {
				t.Error("creation estimate gained a recipient")
			}
			reply := map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": "0x10000"}
			if refused {
				delete(reply, "result")
				reply["error"] = map[string]any{"code": -32000, "message": "test estimation refused"}
			}
			_ = json.NewEncoder(w).Encode(reply)
		}))
		d := deps()
		ac := &interp.ActionCtx{Env: envWithNode(t, server.URL), Deps: &d, Args: map[string]any{"nonce": "7", "gasPrice": "3"}}
		gas, err := localAssetGas(context.Background(), ac, testTo, "", big.NewInt(5), "0x6000")
		server.Close()
		if refused {
			if gas != "" || err == nil || !strings.Contains(err.Error(), "test estimation refused") {
				t.Fatalf("estimation failure became a default: %q %v", gas, err)
			}
		} else if err != nil || gas != "0x10000" {
			t.Fatalf("estimated gas changed: %q %v", gas, err)
		}
	}
}
