package accounts_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the case

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// TestGenesisBalanceCase_RegistersAndPasses validates the tests/ convention end
// to end: the case registers on import and passes against a mock node that
// reports a non-zero balance for the funded account.
func TestGenesisBalanceCase_RegistersAndPasses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		result := "0x0"
		if req.Method == "eth_getBalance" {
			result = "0x84595161401484a000000"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	defer srv.Close()

	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "genesis-balance" {
			found = true
		}
	}
	if !found {
		t.Fatal("genesis-balance case not registered on import")
	}

	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: srv.URL}})
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"genesis-balance"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusPass {
		t.Fatalf("genesis-balance result: %+v", rep.Results)
	}
}

// TestGenesisBalanceCase_FailsWhenUnfunded confirms the case fails when the
// account has a zero balance (alloc not applied).
func TestGenesisBalanceCase_FailsWhenUnfunded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x0"})
	}))
	defer srv.Close()

	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: srv.URL}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"genesis-balance"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusFail {
		t.Fatalf("expected fail for zero balance, got %+v", rep.Results)
	}
}
