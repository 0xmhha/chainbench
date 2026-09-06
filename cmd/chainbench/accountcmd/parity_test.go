package accountcmd_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/accountcmd"
	"github.com/0xmhha/chainbench/internal/mcp"
)

// These tests exist because this package exists. Until the account verbs moved
// out of package main they could not be called from a test at all, so the CLI
// and the MCP tool that answer the same question had never been compared. That
// is what U1 is for (worklist §1l).

// node is a stand-in chain that answers a fixed set of RPC methods and records
// what it was asked. The record is the interesting part: two surfaces that ask
// a node the same questions are doing the same thing, whatever they print.
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

// asked returns the methods seen so far and forgets them, so each surface's
// questions are collected separately.
func (n *node) asked() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := append([]string(nil), n.seen...)
	n.seen = nil
	return out
}

// runCLI executes an account command line the way an operator types it, on a
// bare root, so the test exercises what the real root mounts without depending
// on package main.
func runCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(accountcmd.New(), accountcmd.NewFaucet())
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

// runMCP calls an MCP tool and returns its text content.
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
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("%s: bad response: %v (%s)", tool, err, raw)
	}
	if resp.Error != nil {
		t.Fatalf("%s: %s", tool, resp.Error.Message)
	}
	if len(resp.Result.Content) == 0 {
		t.Fatalf("%s: no content in %s", tool, raw)
	}
	if resp.Result.IsError {
		t.Fatalf("%s failed: %s", tool, resp.Result.Content[0].Text)
	}
	return resp.Result.Content[0].Text
}

// TestParity_AccountState: the CLI and the MCP tool ask the chain the same
// questions and report the same answers.
//
// The comparison is on what each surface asked the node, not on what it
// printed, because a CLI writes for a person and MCP writes for a program.
// Pinning the text would pin the rendering; pinning the questions pins the
// behaviour. The values are then checked to appear in both renderings, so a
// surface cannot ask correctly and report nonsense.
//
// Both reach core/rpc directly today. That is the divergence risk U4 removes,
// and until it does, this is what notices if one of them drifts.
func TestParity_AccountState(t *testing.T) {
	const (
		wei   = "0xde0b6b3a7640000" // 1 ether
		nonce = "0x2"
		code  = "0x6001"
	)
	n := newNode(t, map[string]any{
		"eth_getBalance": wei, "eth_getTransactionCount": nonce, "eth_getCode": code,
	})
	const addr = "0xabc"

	cliOut, err := runCLI(t, "account", "state", "--rpc", n.URL, "--address", addr)
	if err != nil {
		t.Fatalf("CLI: %v\n%s", err, cliOut)
	}
	cliAsked := n.asked()

	mcpOut := runMCP(t, "chainbench_account_state", map[string]any{"rpc": n.URL, "address": addr})
	mcpAsked := n.asked()

	if len(cliAsked) == 0 {
		t.Fatal("the CLI asked the node nothing, so agreeing about it proves nothing")
	}
	if !reflect.DeepEqual(cliAsked, mcpAsked) {
		t.Errorf("the two surfaces ask the chain different questions.\n  CLI: %v\n  MCP: %v", cliAsked, mcpAsked)
	}
	// The same facts have to survive into both renderings, however each is laid
	// out: one ether in wei, nonce 2, and code present.
	for _, want := range []string{"1000000000000000000", "2"} {
		for name, out := range map[string]string{"CLI": cliOut, "MCP": mcpOut} {
			if !strings.Contains(out, want) {
				t.Errorf("%s output lost %q:\n%s", name, want, out)
			}
		}
	}
	if !strings.Contains(cliOut, "contract: true") || !strings.Contains(mcpOut, "contract=true") {
		t.Errorf("the surfaces disagree about the account holding code.\n  CLI: %s\n  MCP: %s", cliOut, mcpOut)
	}
}

// TestAccountState_RefusesWithoutATarget: the command says what is missing
// rather than dialling an empty URL.
func TestAccountState_RefusesWithoutATarget(t *testing.T) {
	for _, args := range [][]string{
		{"account", "state", "--address", "0xabc"},
		{"account", "state", "--rpc", "http://127.0.0.1:1"},
	} {
		out, err := runCLI(t, args...)
		if err == nil {
			t.Errorf("%v was accepted with a missing flag:\n%s", args, out)
			continue
		}
		if !strings.Contains(err.Error(), "--rpc and --address are required") {
			t.Errorf("%v: the error does not say what is missing: %v", args, err)
		}
	}
}

// TestFaucet_RejectsANonDecimalAmount: wei is decimal, and a hex amount that
// silently parsed as something else would move the wrong sum.
func TestFaucet_RejectsANonDecimalAmount(t *testing.T) {
	out, err := runCLI(t, "faucet", "--rpc", "http://127.0.0.1:1", "--from-key", "0x01",
		"--to", "0xabc", "--amount", "0x10")
	if err == nil {
		t.Fatalf("a hex amount was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "decimal wei expected") {
		t.Errorf("the error does not say what the amount should look like: %v", err)
	}
}

// TestParity_Faucet: funding is the verb a test harness leans on most — a
// generated account cannot pay for its own first transaction — so the two
// surfaces must move the same sum from the same key.
//
// Comparing the signed transaction covers the amount, the recipient and the
// sender at once. Both surfaces went through their own copy of this until U4
// gave it one entry point.
func TestParity_Faucet(t *testing.T) {
	n := newSigningNode(t)
	const (
		key    = "0xeb47b675926a348755d89dfaca9ba5a2c02a192fd54e7e78475f15443ddf8c21"
		to     = "0x000000000000000000000000000000000000dEaD"
		amount = "1000000000000000000"
	)

	cli, err := runCLI(t, "faucet", "--rpc", n.URL, "--from-key", key, "--to", to, "--amount", amount)
	if err != nil {
		t.Fatalf("CLI faucet: %v\n%s", err, cli)
	}
	cliSent := n.broadcast()

	runMCP(t, "chainbench_faucet", map[string]any{
		"rpc": n.URL, "from_key": key, "to": to, "amount": amount,
	})
	mcpSent := n.broadcast()

	if len(cliSent) != 1 {
		t.Fatalf("the CLI broadcast %d transactions, want 1 — nothing to compare", len(cliSent))
	}
	if len(mcpSent) != 1 || cliSent[0] != mcpSent[0] {
		t.Errorf("the two surfaces funded with different transactions.\n  CLI: %v\n  MCP: %v", cliSent, mcpSent)
	}
}

// signingNode answers the calls a wallet makes on its way to broadcasting, and
// records the raw transaction it was given.
type signingNode struct {
	*httptest.Server
	mu   sync.Mutex
	sent []string
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
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if req.Method == "eth_sendRawTransaction" {
			if len(req.Params) > 0 {
				if raw, ok := req.Params[0].(string); ok {
					n.mu.Lock()
					n.sent = append(n.sent, raw)
					n.mu.Unlock()
				}
			}
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

func (n *signingNode) broadcast() []string {
	n.mu.Lock()
	defer n.mu.Unlock()
	out := append([]string(nil), n.sent...)
	n.sent = nil
	return out
}
