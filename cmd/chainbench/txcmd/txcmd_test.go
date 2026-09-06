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

// U4 gave the on-chain verbs a single entry point in app, and these are what
// that buys: before it, `tx send` was written three times over — here, in the
// MCP tools, and in the DSL's built-in actions — and nothing compared them.

// signingNode answers the calls a wallet makes on its way to broadcasting, so
// a send can be driven end to end without a chain. It records the raw
// transaction each surface produced.
type signingNode struct {
	*httptest.Server
	mu   sync.Mutex
	sent []string
	seen []string
}

func newSigningNode(t *testing.T) *signingNode {
	t.Helper()
	n := &signingNode{}
	results := map[string]any{
		"eth_chainId":              "0x205b",
		"eth_getTransactionCount":  "0x0",
		"eth_gasPrice":             "0x3b9aca00",
		"eth_maxPriorityFeePerGas": "0x3b9aca00",
		"eth_estimateGas":          "0x5208",
		// stablenet asks whether the sender is blacklisted before it signs, and
		// an empty proof is how a plain account answers.
		"eth_getProof": map[string]any{
			"balance": "0x0", "nonce": "0x0", "codeHash": "0x", "storageProof": []any{},
		},
		"eth_getBlockByNumber": map[string]any{
			"number": "0x1", "baseFeePerGas": "0x7", "gasLimit": "0x1c9c380",
		},
	}
	n.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		n.mu.Lock()
		n.seen = append(n.seen, req.Method)
		if req.Method == "eth_sendRawTransaction" && len(req.Params) > 0 {
			if raw, ok := req.Params[0].(string); ok {
				n.sent = append(n.sent, raw)
			}
		}
		n.mu.Unlock()
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if req.Method == "eth_sendRawTransaction" {
			resp["result"] = "0x" + strings.Repeat("ab", 32)
		} else if v, ok := results[req.Method]; ok {
			resp["result"] = v
		} else {
			resp["error"] = map[string]any{"code": -32601, "message": "method not found"}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(n.Close)
	return n
}

// broadcast returns the raw transactions seen so far and forgets them.
func (n *signingNode) broadcast() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := append([]string(nil), n.sent...)
	n.sent = nil
	n.seen = nil
	return out
}

const testKey = "0xeb47b675926a348755d89dfaca9ba5a2c02a192fd54e7e78475f15443ddf8c21"

// TestParity_TxSend: the same request signed by either surface has to produce
// the same transaction, byte for byte.
//
// Comparing the raw signed transaction is the strongest check available here:
// it covers the nonce, the gas fields, the value, the calldata and the
// signature at once. Two surfaces that agree on this cannot be reading --value
// or --data differently.
func TestParity_TxSend(t *testing.T) {
	n := newSigningNode(t)
	const to, data, value = "0x000000000000000000000000000000000000dEaD", "0xdeadbeef", "1000"

	cli, err := runCLI(t, "tx", "send", "--rpc", n.URL, "--from-key", testKey,
		"--to", to, "--data", data, "--value", value)
	if err != nil {
		t.Fatalf("CLI tx send: %v\n%s", err, cli)
	}
	cliSent := n.broadcast()

	runMCP(t, "chainbench_tx_send", map[string]any{
		"rpc": n.URL, "from_key": testKey, "to": to, "data": data, "value": value,
	})
	mcpSent := n.broadcast()

	if len(cliSent) != 1 {
		t.Fatalf("the CLI broadcast %d transactions, want 1 — nothing to compare", len(cliSent))
	}
	if len(mcpSent) != 1 || cliSent[0] != mcpSent[0] {
		t.Errorf("the two surfaces signed different transactions.\n  CLI: %v\n  MCP: %v", cliSent, mcpSent)
	}
}

// TestParity_ContractDeploy: same, for a creation transaction — the one whose
// value both surfaces default and whose bytecode both must read as hex.
func TestParity_ContractDeploy(t *testing.T) {
	n := newSigningNode(t)
	const code = "0x6080604052"

	cli, err := runCLI(t, "contract", "deploy", "--rpc", n.URL, "--from-key", testKey, "--bytecode", code)
	if err != nil {
		t.Fatalf("CLI contract deploy: %v\n%s", err, cli)
	}
	cliSent := n.broadcast()

	runMCP(t, "chainbench_contract_deploy", map[string]any{
		"rpc": n.URL, "from_key": testKey, "bytecode": code,
	})
	mcpSent := n.broadcast()

	if len(cliSent) != 1 {
		t.Fatalf("the CLI broadcast %d transactions, want 1 — nothing to compare", len(cliSent))
	}
	if len(mcpSent) != 1 || cliSent[0] != mcpSent[0] {
		t.Errorf("the two surfaces deployed different transactions.\n  CLI: %v\n  MCP: %v", cliSent, mcpSent)
	}
}

// TestParity_TxWait: a receipt read by either surface reports the same facts.
// The renderings differ — the CLI lays the receipt out for a person, MCP
// answers with it as JSON — so the facts are what is compared.
func TestParity_TxWait(t *testing.T) {
	const hash = "0xabc"
	n := newNode(t, map[string]any{
		"eth_getTransactionReceipt": map[string]any{
			"status": "0x1", "blockNumber": "0x2a", "gasUsed": "0x5208",
		},
	})

	cli, err := runCLI(t, "tx", "wait", "--rpc", n.URL, "--hash", hash)
	if err != nil {
		t.Fatalf("CLI tx wait: %v\n%s", err, cli)
	}
	mcpOut := runMCP(t, "chainbench_tx_wait", map[string]any{"rpc": n.URL, "hash": hash})

	var r struct {
		Status      string `json:"status"`
		BlockNumber string `json:"blockNumber"`
		GasUsed     string `json:"gasUsed"`
	}
	if err := json.Unmarshal([]byte(mcpOut), &r); err != nil {
		t.Fatalf("the tool's answer is not a receipt: %v\n%s", err, mcpOut)
	}
	if r.Status == "" || r.BlockNumber == "" {
		t.Fatalf("the tool reported an empty receipt, so agreeing about it proves nothing:\n%s", mcpOut)
	}
	// The CLI spells a status out for a person, so the block and the gas are
	// what can be compared verbatim; the status is checked by its meaning.
	for _, want := range []string{r.BlockNumber, r.GasUsed} {
		if !strings.Contains(cli, want) {
			t.Errorf("the CLI's receipt lost %q:\n%s", want, cli)
		}
	}
	if !strings.Contains(cli, "success") {
		t.Errorf("the tool reported status %s; the CLI does not call it a success:\n%s", r.Status, cli)
	}
}
