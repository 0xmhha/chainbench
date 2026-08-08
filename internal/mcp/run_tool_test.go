package mcp_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// mockRPCNode serves canned JSON-RPC results keyed by method.
func mockRPCNode(t *testing.T, results map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if v, ok := results[req.Method]; ok {
			res["result"] = v
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func specJSON(t *testing.T, spec map[string]any) string {
	t.Helper()
	b, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRunTool_AttachMockRPC(t *testing.T) {
	srv := mockRPCNode(t, map[string]any{
		"eth_chainId":     "0x539", // 1337
		"eth_blockNumber": "0x10",  // 16
	})
	spec := specJSON(t, map[string]any{
		"schemaVersion": "1",
		"id":            "mcp-smoke",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{
			{"assert": "chainId", "expected": 1337},
			{"assert": "blockNumber", "expected": 1},
		},
	})

	text, isErr := callText(t, newServer(), "chainbench_run", map[string]any{
		"chain": "stablenet",
		"rpc":   []any{srv.URL},
		"spec":  spec,
	})
	if isErr {
		t.Fatalf("chainbench_run reported error: %s", text)
	}
	if !strings.Contains(text, "pass=1") || !strings.Contains(text, "mcp-smoke") {
		t.Fatalf("unexpected run output:\n%s", text)
	}
}

func TestRunTool_MissingArgs(t *testing.T) {
	// No spec.
	text, isErr := callText(t, newServer(), "chainbench_run", map[string]any{
		"chain": "stablenet",
		"rpc":   []any{"http://127.0.0.1:1"},
	})
	if !isErr {
		t.Fatalf("expected error when no spec is given, got: %s", text)
	}
}
