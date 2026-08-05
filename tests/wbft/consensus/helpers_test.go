package consensus_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// rpcRequest is the subset of a JSON-RPC request the mocks route on.
type rpcRequest struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params []any  `json:"params"`
}

// mockServer answers JSON-RPC POSTs via handle(method, params) -> result.
func mockServer(t *testing.T, handle func(method string, params []any) any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req rpcRequest
		_ = json.Unmarshal(body, &req)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": req.ID, "result": handle(req.Method, req.Params),
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// consensusSet builds a wbft NodeSet advertising the consensus capability, with
// one node per (role, rpcURL) pair, indexed from 1.
func consensusSet(nodes ...node.Node) node.NodeSet {
	for i := range nodes {
		nodes[i].Index = i + 1
	}
	return node.NodeSet{
		Chain:        "wbft",
		Network:      "local",
		Capabilities: []string{"rpc", "consensus"},
		Nodes:        nodes,
	}
}

// runCase runs a single named case against ns and returns its result.
func runCase(t *testing.T, ns node.NodeSet, name string) testkit.Result {
	t.Helper()
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{name}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 {
		t.Fatalf("%s: want 1 result, got %+v", name, rep.Results)
	}
	return rep.Results[0]
}

// registered reports whether a case with the given name is registered.
func registered(name string) bool {
	for _, c := range testkit.Cases() {
		if c.Name == name {
			return true
		}
	}
	return false
}
