package consensus_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/consensus" // register the case

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// TestChainIDCase_RegistersAndPasses validates the tests/ convention end to
// end: the case registers on import and passes against a mock node returning a
// stable chain id.
func TestChainIDCase_RegistersAndPasses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x205b"})
	}))
	defer srv.Close()

	// Confirm the case registered on import.
	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "chain-id" {
			found = true
		}
	}
	if !found {
		t.Fatal("chain-id case not registered on import")
	}

	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: srv.URL}})
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"chain-id"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusPass {
		t.Fatalf("chain-id result: %+v", rep.Results)
	}
}

// TestChainIDCase_SkipsForeignChain confirms chain_compat gating: the case does
// not run on a chain outside its ChainCompat.
func TestChainIDCase_SkipsForeignChain(t *testing.T) {
	ns, _ := attach.Build("ethereum", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"chain-id"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on foreign chain, got %+v", rep.Results)
	}
}
