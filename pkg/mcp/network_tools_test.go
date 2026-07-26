package mcp_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// probeMock answers mapped methods with a result and every other method with a
// JSON-RPC method-not-found error, so probe's namespace detection is accurate
// (the shared rpcMock returns a null result for unmapped methods, which probe
// would read as the method being supported).
func probeMock(results map[string]any) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if v, ok := results[req.Method]; ok {
			resp["result"] = v
		} else {
			resp["error"] = map[string]any{"code": -32601, "message": "method not found"}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// TestNetworkTools exercises the attach -> list -> info -> detach lifecycle,
// backed by the core probe (chain detection) and named-network registry.
func TestNetworkTools(t *testing.T) {
	// endpoint that identifies as stablenet: chain id 8283 + istanbul namespace.
	srv := probeMock(map[string]any{
		"eth_chainId":            "0x205b", // 8283
		"istanbul_getValidators": []any{},
	})
	defer srv.Close()
	dir := t.TempDir()
	s := newServer()

	text, isErr := callText(t, s, "chainbench_network_attach", map[string]any{
		"name": "prod", "rpc": srv.URL, "state_dir": dir,
	})
	if isErr || !strings.Contains(text, "chain_type=stablenet") || !strings.Contains(text, "chain_id=8283") {
		t.Fatalf("attach: err=%v text=%s", isErr, text)
	}

	text, isErr = callText(t, s, "chainbench_network_list", map[string]any{"state_dir": dir})
	if isErr || !strings.Contains(text, "prod") || !strings.Contains(text, "stablenet") {
		t.Errorf("list: err=%v text=%s", isErr, text)
	}

	text, isErr = callText(t, s, "chainbench_network_info", map[string]any{"name": "prod", "state_dir": dir})
	if isErr || !strings.Contains(text, "network=prod") || !strings.Contains(text, srv.URL) {
		t.Errorf("info: err=%v text=%s", isErr, text)
	}

	text, isErr = callText(t, s, "chainbench_network_detach", map[string]any{"name": "prod", "state_dir": dir})
	if isErr || !strings.Contains(text, "detached") {
		t.Errorf("detach: err=%v text=%s", isErr, text)
	}
	// after detach the list is empty
	text, _ = callText(t, s, "chainbench_network_list", map[string]any{"state_dir": dir})
	if !strings.Contains(text, "no attached networks") {
		t.Errorf("post-detach list should be empty: %s", text)
	}
}

func TestNetworkAttach_Rejects(t *testing.T) {
	dir := t.TempDir()
	s := newServer()
	// reserved name is rejected before any probe.
	_, isErr := callText(t, s, "chainbench_network_attach", map[string]any{
		"name": "local", "rpc": "http://x", "state_dir": dir,
	})
	if !isErr {
		t.Error("reserved name 'local' should be rejected")
	}
	// missing args.
	if _, isErr := callText(t, s, "chainbench_network_attach", map[string]any{"name": "n"}); !isErr {
		t.Error("missing rpc/state_dir should error")
	}
}
