package txcmd_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/txcmd"
	"github.com/0xmhha/chainbench/internal/mcp"
)

// The tx and contract verbs had no tests while they lived in package main,
// because a test cannot call into package main. Moving them here is what makes
// these possible, which is the whole of U1 (worklist §1l).

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
	root.AddCommand(txcmd.NewTx(), txcmd.NewContract())
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
	if len(resp.Result.Content) == 0 {
		t.Fatalf("%s: no content in %s", tool, raw)
	}
	if resp.Result.IsError {
		t.Fatalf("%s failed: %s", tool, resp.Result.Content[0].Text)
	}
	return resp.Result.Content[0].Text
}

// TestParity_ContractCall: both surfaces put the same question to the chain and
// report the same answer. A read-only call is the one contract verb that needs
// no key, which is what lets it be compared here rather than on a live chain.
func TestParity_ContractCall(t *testing.T) {
	n := newNode(t, map[string]any{"eth_call": "0x2a"})
	const to, data = "0xdead", "0xabcd"

	cliOut, err := runCLI(t, "contract", "call", "--rpc", n.URL, "--to", to, "--data", data)
	if err != nil {
		t.Fatalf("CLI: %v\n%s", err, cliOut)
	}
	cliAsked := n.asked()

	mcpOut := runMCP(t, "chainbench_contract_call", map[string]any{"rpc": n.URL, "to": to, "data": data})
	mcpAsked := n.asked()

	if len(cliAsked) == 0 {
		t.Fatal("the CLI asked the node nothing, so agreeing about it proves nothing")
	}
	if strings.Join(cliAsked, ",") != strings.Join(mcpAsked, ",") {
		t.Errorf("the two surfaces ask the chain different questions.\n  CLI: %v\n  MCP: %v", cliAsked, mcpAsked)
	}
	for name, out := range map[string]string{"CLI": cliOut, "MCP": mcpOut} {
		if !strings.Contains(out, "0x2a") {
			t.Errorf("%s lost the call result:\n%s", name, out)
		}
	}
}

// TestSend_RefusesWithoutATarget: the command names what is missing instead of
// signing against an empty URL.
func TestSend_RefusesWithoutATarget(t *testing.T) {
	out, err := runCLI(t, "tx", "send", "--to", "0xabc")
	if err == nil {
		t.Fatalf("a send with no rpc or key was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--rpc, --from-key and --to are required") {
		t.Errorf("the error does not say what is missing: %v", err)
	}
}

// TestSend_RejectsABadKey: a key that is not hex has to fail before anything is
// signed, and the message has to say which flag was wrong.
func TestSend_RejectsABadKey(t *testing.T) {
	out, err := runCLI(t, "tx", "send", "--rpc", "http://127.0.0.1:1",
		"--from-key", "zz", "--to", "0xabc")
	if err == nil {
		t.Fatalf("a non-hex key was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "bad --from-key") {
		t.Errorf("the error does not name the bad flag: %v", err)
	}
}

// TestSend_RejectsANonDecimalValue: wei is decimal here, and a hex value that
// parsed as something else would move the wrong sum.
func TestSend_RejectsANonDecimalValue(t *testing.T) {
	out, err := runCLI(t, "tx", "send", "--rpc", "http://127.0.0.1:1",
		"--from-key", "0x01", "--to", "0xabc", "--value", "0x10")
	if err == nil {
		t.Fatalf("a hex value was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "decimal wei expected") {
		t.Errorf("the error does not say what the value should look like: %v", err)
	}
}

// TestDeploy_RefusesWithoutBytecode: there is nothing to deploy without it, and
// the failure belongs before the wallet is opened.
func TestDeploy_RefusesWithoutBytecode(t *testing.T) {
	out, err := runCLI(t, "contract", "deploy", "--rpc", "http://127.0.0.1:1", "--from-key", "0x01")
	if err == nil {
		t.Fatalf("a deploy with no bytecode was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--bytecode") {
		t.Errorf("the error does not mention the missing bytecode: %v", err)
	}
}
