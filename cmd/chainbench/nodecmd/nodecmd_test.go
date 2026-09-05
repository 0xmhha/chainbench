package nodecmd_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/nodecmd"
	"github.com/0xmhha/chainbench/internal/mcp"
)

// node answers a fixed set of RPC methods and records what it was asked, so two
// surfaces can be compared on the questions they put to a chain rather than on
// what they print.
type node struct {
	*httptest.Server
	mu   sync.Mutex
	seen []string
}

func newNode(t *testing.T, results map[string]any) *node {
	t.Helper()
	n := &node{}
	n.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		n.mu.Lock()
		n.seen = append(n.seen, req.Method)
		n.mu.Unlock()
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if v, ok := results[req.Method]; ok {
			resp["result"] = v
		} else {
			resp["error"] = map[string]any{"code": -32601, "message": "method not found"}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(n.Close)
	return n
}

func (n *node) asked() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := append([]string(nil), n.seen...)
	n.seen = nil
	return out
}

func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(nodecmd.New())
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

func runMCP(t *testing.T, tool string, args map[string]any) string {
	t.Helper()
	req, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": args},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := mcp.Default("chainbench", "test").Handle(context.Background(), req)
	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("%s: bad response: %v (%s)", tool, err, raw)
	}
	if len(resp.Result.Content) == 0 || resp.Result.IsError {
		t.Fatalf("%s failed: %s", tool, raw)
	}
	return resp.Result.Content[0].Text
}

// TestParity_NodeRPC: a raw RPC passthrough exists so an operator can ask a node
// anything. Both surfaces offer it, so both must forward the same call.
func TestParity_NodeRPC(t *testing.T) {
	n := newNode(t, map[string]any{"eth_blockNumber": "0x2a"})

	cli, err := runCLI(t, "node", "rpc", "--rpc", n.URL, "--method", "eth_blockNumber")
	if err != nil {
		t.Fatalf("CLI: %v\n%s", err, cli)
	}
	cliAsked := n.asked()

	mcpOut := runMCP(t, "chainbench_node_rpc", map[string]any{"rpc": n.URL, "method": "eth_blockNumber"})
	mcpAsked := n.asked()

	if len(cliAsked) == 0 {
		t.Fatal("the CLI forwarded nothing, so agreeing proves nothing")
	}
	if strings.Join(cliAsked, ",") != strings.Join(mcpAsked, ",") {
		t.Errorf("the two surfaces forward different calls.\n  CLI: %v\n  MCP: %v", cliAsked, mcpAsked)
	}
	for name, out := range map[string]string{"CLI": cli, "MCP": mcpOut} {
		if !strings.Contains(out, "0x2a") {
			t.Errorf("%s lost the node's answer:\n%s", name, out)
		}
	}
}

// TestNodeRPC_ReportsTheNodesError: an RPC error belongs to the operator, not
// swallowed into an empty result.
func TestNodeRPC_ReportsTheNodesError(t *testing.T) {
	n := newNode(t, map[string]any{})
	out, err := runCLI(t, "node", "rpc", "--rpc", n.URL, "--method", "eth_nonesuch")
	if err == nil && !strings.Contains(out, "method not found") {
		t.Fatalf("an unknown method produced neither an error nor the node's message:\n%s", out)
	}
}

// TestNodeStop_RefusesWithoutAWorkspace: the pids come from a workspace, so
// without one there is nothing to stop and the command has to say so rather
// than report success.
func TestNodeStop_RefusesWithoutAWorkspace(t *testing.T) {
	out, err := runCLI(t, "node", "stop", "--workspace-dir", t.TempDir(), "--node", "1")
	if err == nil {
		t.Fatalf("stopping a node in an empty workspace was reported as done:\n%s", out)
	}
}
