package anzeon_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the case

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func TestNativeCoinAdapterCodeCase(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var result any
		if req.Method == "eth_getCode" {
			result = "0x60806040" // non-empty deployed code
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	defer srv.Close()

	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: srv.URL}})
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"native-coin-adapter-code"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusPass {
		t.Fatalf("native-coin-adapter-code: %+v", rep.Results)
	}
}

func TestNativeCoinAdapterCodeCase_SkipsForeignChain(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"native-coin-adapter-code"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wbft, got %+v", rep.Results)
	}
}
