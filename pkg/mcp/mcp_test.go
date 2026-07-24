package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/pkg/chains/all"
	_ "github.com/0xmhha/chainbench/tests/all"

	"github.com/0xmhha/chainbench/pkg/mcp"
)

func newServer() *mcp.Server { return mcp.Default("chainbench", "test") }

// call sends a JSON-RPC request to the server and returns the decoded response.
func call(t *testing.T, s *mcp.Server, req map[string]any) map[string]any {
	t.Helper()
	b, _ := json.Marshal(req)
	out := s.Handle(context.Background(), b)
	if out == nil {
		return nil
	}
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("bad response: %v (%s)", err, out)
	}
	return resp
}

// callText invokes a tool and returns the text content.
func callText(t *testing.T, s *mcp.Server, name string, args map[string]any) (string, bool) {
	t.Helper()
	resp := call(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 9, "method": "tools/call",
		"params": map[string]any{"name": name, "arguments": args},
	})
	result := resp["result"].(map[string]any)
	content := result["content"].([]any)
	text := content[0].(map[string]any)["text"].(string)
	isErr, _ := result["isError"].(bool)
	return text, isErr
}

func TestInitializeAndList(t *testing.T) {
	s := newServer()
	initResp := call(t, s, map[string]any{"jsonrpc": "2.0", "id": 1, "method": "initialize"})
	si := initResp["result"].(map[string]any)["serverInfo"].(map[string]any)
	if si["name"] != "chainbench" {
		t.Errorf("serverInfo: %v", si)
	}

	listResp := call(t, s, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
	tools := listResp["result"].(map[string]any)["tools"].([]any)
	if len(tools) != 6 {
		t.Fatalf("tools: %d, want 6", len(tools))
	}
	names := map[string]bool{}
	for _, tt := range tools {
		names[tt.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{"chainbench_chains", "chainbench_faucet", "chainbench_verify", "chainbench_test", "chainbench_consensus", "chainbench_node_rpc"} {
		if !names[want] {
			t.Errorf("missing tool %q", want)
		}
	}
}

func TestNodeRPCTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID int `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x2a"})
	}))
	defer srv.Close()
	text, isErr := callText(t, newServer(), "chainbench_node_rpc", map[string]any{
		"rpc": srv.URL, "method": "eth_blockNumber",
	})
	if isErr || text != `"0x2a"` {
		t.Errorf("node_rpc tool: err=%v text=%s", isErr, text)
	}
}

func TestConsensusTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if req.Method == "istanbul_getValidators" {
			res["result"] = []string{"0xaaa", "0xbbb"}
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()
	text, isErr := callText(t, newServer(), "chainbench_consensus", map[string]any{"chain": "stablenet", "rpc": srv.URL})
	if isErr || !strings.Contains(text, "istanbul_getValidators") || !strings.Contains(text, "0xaaa") {
		t.Errorf("consensus tool: err=%v text=%s", isErr, text)
	}
}

func TestNotificationNoResponse(t *testing.T) {
	s := newServer()
	if out := s.Handle(context.Background(), []byte(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)); out != nil {
		t.Errorf("notification should have no response, got %s", out)
	}
}

func TestChainsTool(t *testing.T) {
	text, isErr := callText(t, newServer(), "chainbench_chains", map[string]any{})
	if isErr || !strings.Contains(text, "stablenet") || !strings.Contains(text, "wemix") {
		t.Errorf("chains tool: err=%v text=%s", isErr, text)
	}
}

func TestUnknownTool(t *testing.T) {
	resp := call(t, newServer(), map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "tools/call",
		"params": map[string]any{"name": "nope", "arguments": map[string]any{}},
	})
	if _, ok := resp["error"]; !ok {
		t.Errorf("expected error for unknown tool: %v", resp)
	}
}

func mockNode(t *testing.T) *httptest.Server {
	t.Helper()
	var block int
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		switch req.Method {
		case "eth_blockNumber":
			block++
			res["result"] = "0x" + hexInt(block)
		case "eth_chainId":
			res["result"] = "0x205b"
		case "net_peerCount":
			res["result"] = "0x2"
		case "eth_syncing":
			res["result"] = false
		case "eth_getProof":
			res["result"] = map[string]any{"extra": "0x0"}
		case "eth_getTransactionCount":
			res["result"] = "0x0"
		case "eth_maxPriorityFeePerGas", "eth_gasPrice":
			res["result"] = "0x3b9aca00"
		case "eth_sendRawTransaction":
			res["result"] = "0x" + strings.Repeat("ab", 32)
		default:
			res["result"] = nil
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
}

func TestVerifyAndTestTools(t *testing.T) {
	srv := mockNode(t)
	defer srv.Close()
	s := newServer()

	vtext, verr := callText(t, s, "chainbench_verify", map[string]any{"chain": "wbft", "rpc": []any{srv.URL}})
	if verr || !strings.Contains(vtext, "producing: true") {
		t.Errorf("verify tool: err=%v text=%s", verr, vtext)
	}

	ttext, terr := callText(t, s, "chainbench_test", map[string]any{
		"chain": "wbft", "rpc": []any{srv.URL}, "name": []any{"chain-id"},
	})
	if terr || !strings.Contains(ttext, "pass=1") {
		t.Errorf("test tool: err=%v text=%s", terr, ttext)
	}
}

func TestFaucetTool(t *testing.T) {
	srv := mockNode(t)
	defer srv.Close()
	text, isErr := callText(t, newServer(), "chainbench_faucet", map[string]any{
		"chain":    "stablenet",
		"rpc":      srv.URL,
		"from_key": "0000000000000000000000000000000000000000000000000000000000000001",
		"to":       "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf",
		"amount":   "1000000",
	})
	if isErr || !strings.HasPrefix(text, "tx: 0x") {
		t.Errorf("faucet tool: err=%v text=%s", isErr, text)
	}
}

func hexInt(n int) string {
	const d = "0123456789abcdef"
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{d[n%16]}, b...)
		n /= 16
	}
	return string(b)
}
