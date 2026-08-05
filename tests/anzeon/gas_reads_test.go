package anzeon_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// gasMock answers the gas-API reads with a self-consistent set of values:
// baseFee=0x64 (100), gasTip=0xa (10) so gasPrice=0x6e (110) == 100+10, and
// maxPriorityFee=0xa == gasTip. estimateGas=0x8000 (32768) is well over 21000.
func gasMock(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		var result any
		switch req.Method {
		case "eth_gasPrice":
			result = "0x6e" // 110 == baseFee(100) + gasTip(10)
		case "eth_maxPriorityFeePerGas":
			result = "0xa" // 10 == gasTip
		case "eth_getBlockByNumber":
			result = map[string]any{"number": "0x10", "baseFeePerGas": "0x64"}
		case "istanbul_getWbftExtraInfo":
			result = map[string]any{"gasTip": "0xa"}
		case "eth_estimateGas":
			result = "0x8000" // 32768 > 21000
		default:
			result = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGasReadCases(t *testing.T) {
	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: gasMock(t).URL}})
	for _, name := range []string{
		"gas-price-equals-basefee-plus-tip",
		"max-priority-fee-equals-gastip",
		"estimate-gas-token-transfer",
	} {
		rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{name}})
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusPass {
			t.Errorf("%s: %+v", name, rep.Results)
		}
	}
}

// gasMismatchMock returns a gasPrice that does not equal baseFee+gasTip, so the
// gas-price case must fail.
func gasMismatchMock(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		var result any
		switch req.Method {
		case "eth_gasPrice":
			result = "0x1" // != 110
		case "eth_getBlockByNumber":
			result = map[string]any{"number": "0x10", "baseFeePerGas": "0x64"}
		case "istanbul_getWbftExtraInfo":
			result = map[string]any{"gasTip": "0xa"}
		default:
			result = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGasPriceCase_MismatchFails(t *testing.T) {
	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: gasMismatchMock(t).URL}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"gas-price-equals-basefee-plus-tip"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusFail {
		t.Fatalf("mismatched gasPrice should fail: %+v", rep.Results)
	}
}

func TestGasReadCases_SkipForeignChain(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"gas-price-equals-basefee-plus-tip"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wbft, got %+v", rep.Results)
	}
}
