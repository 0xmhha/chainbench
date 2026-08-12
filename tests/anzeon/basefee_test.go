package anzeon_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the case

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// blockMock answers eth_getBlockByNumber with a base fee at or above the anzeon
// minimum (0x1236efcbcbb340000 is well over 2e13).
func blockMock(t *testing.T, baseFeeHex string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var result any
		if req.Method == "eth_getBlockByNumber" {
			result = map[string]any{"baseFeePerGas": baseFeeHex}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestBasefeeMinimumCase(t *testing.T) {
	// 0x2e90edd000 = 200e9, comfortably above the 2e13... no: use a value above
	// the 20000000000000 (2e13) minimum. 0x2d79883d2000 = 5e13.
	ns, _ := node.AttachedSet("stablenet", "local", []node.RPCEndpoint{{RPCURL: blockMock(t, "0x2d79883d2000").URL}})
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"basefee-minimum"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusPass {
		t.Fatalf("basefee-minimum: %+v", rep.Results)
	}
}

func TestBasefeeMaximumCase(t *testing.T) {
	// 5e13 is within [2e13 floor, 2e16 ceiling].
	ns, _ := node.AttachedSet("stablenet", "local", []node.RPCEndpoint{{RPCURL: blockMock(t, "0x2d79883d2000").URL}})
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"basefee-maximum"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusPass {
		t.Fatalf("basefee-maximum: %+v", rep.Results)
	}
}

func TestBasefeeMaximumCase_AboveCeilingFails(t *testing.T) {
	// 0x1000000000000000 = ~1.15e18, above the 2e16 anzeon ceiling.
	ns, _ := node.AttachedSet("stablenet", "local", []node.RPCEndpoint{{RPCURL: blockMock(t, "0x1000000000000000").URL}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"basefee-maximum"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusFail {
		t.Fatalf("above-ceiling base fee should fail: %+v", rep.Results)
	}
}

func TestBasefeeMinimumCase_BelowFloorFails(t *testing.T) {
	// A base fee below the anzeon minimum must fail the case.
	ns, _ := node.AttachedSet("stablenet", "local", []node.RPCEndpoint{{RPCURL: blockMock(t, "0x3b9aca00").URL}}) // 1e9
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"basefee-minimum"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusFail {
		t.Fatalf("below-floor base fee should fail: %+v", rep.Results)
	}
}

func TestBasefeeCase_SkipsForeignChain(t *testing.T) {
	ns, _ := node.AttachedSet("wbft", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"basefee-minimum"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wbft, got %+v", rep.Results)
	}
}
