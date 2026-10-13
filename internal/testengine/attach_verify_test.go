package testengine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestAttachedClientIdentityHonorsBuildAliasAndRejectsWrongNames(t *testing.T) {
	for _, sample := range []struct {
		chain, client string
		accepted      bool
	}{
		{"wbft", "Gwemix/v1.0.0", true},
		{"wbft", "Gwbft/v1.0.0", true},
		{"wemix", "Gwemix/v1.0.0", true},
		{"stablenet", "Gstable/v1.0.0", true},
		{"stablenet", "Gwemix/v1.0.0", false},
		{"wbft", "Gstable/v1.0.0", false},
		{"stablenet", "OtherGstable/v1.0.0", false},
	} {
		t.Run(sample.chain+"/"+sample.client, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ID     any    `json:"id"`
					Method string `json:"method"`
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				if request.Method != "web3_clientVersion" {
					t.Errorf("unexpected probe %s", request.Method)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": sample.client})
			}))
			defer server.Close()
			err := testengine.VerifyAttachedNode(context.Background(), sample.chain, node.Node{RPCURL: server.URL}, false)
			if (err == nil) != sample.accepted {
				t.Fatalf("client %s against %s accepted=%v: %v", sample.client, sample.chain, sample.accepted, err)
			}
		})
	}
}
