package api_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/api" // register the cases

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// apiMock answers the read methods the cases use with self-consistent values.
func apiMock(t *testing.T) *httptest.Server {
	t.Helper()
	const hash = "0xdeadbeef00000000000000000000000000000000000000000000000000000001"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		var result any
		switch req.Method {
		case "eth_getBlockByNumber", "eth_getBlockByHash":
			result = map[string]any{"number": "0x64", "hash": hash, "transactions": []any{}}
		case "eth_gasPrice":
			result = "0x3b9aca00" // 1 gwei
		case "eth_feeHistory":
			result = map[string]any{
				"oldestBlock":   "0x55",
				"baseFeePerGas": []any{"0x12309ce54000", "0x12309ce54000"},
				"gasUsedRatio":  []any{0.0},
			}
		case "txpool_status":
			result = map[string]any{"pending": "0x0", "queued": "0x0"}
		case "txpool_content":
			result = map[string]any{"pending": map[string]any{}, "queued": map[string]any{}}
		case "eth_syncing":
			result = false
		case "eth_estimateGas":
			result = "0x5208" // 21000
		case "eth_getLogs":
			result = []any{map[string]any{
				"address": "0x00000000000000000000000000000000000000ab",
				"topics":  []any{hash},
			}}
		default:
			result = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func run(t *testing.T, name string) testkit.Result {
	t.Helper()
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: apiMock(t).URL}})
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{name}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 {
		t.Fatalf("%s: want 1 result, got %+v", name, rep.Results)
	}
	return rep.Results[0]
}

func TestAPICases(t *testing.T) {
	for _, name := range []string{"block-by-hash-consistency", "gas-price-positive", "txpool-status", "chain-not-syncing", "estimate-gas", "logs-query-well-formed", "block-transactions-field", "fee-history-well-formed", "txpool-content-well-formed"} {
		if r := run(t, name); r.Status != testkit.StatusPass {
			t.Errorf("%s: %+v", name, r)
		}
	}
	// registration + category check
	seen := map[string]string{}
	for _, c := range testkit.Cases() {
		seen[c.Name] = c.Category
	}
	for _, name := range []string{"block-by-hash-consistency", "gas-price-positive"} {
		if seen[name] != "api" {
			t.Errorf("%s category = %q, want api", name, seen[name])
		}
	}
}
