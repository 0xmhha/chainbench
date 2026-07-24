package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// run executes the root command with args and returns combined output.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := newRootCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

func TestChainsCmd(t *testing.T) {
	out, err := run(t, "chains")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"stablenet", "wbft", "wemix", "istanbul", "wemix"} {
		if !strings.Contains(out, want) {
			t.Errorf("chains output missing %q:\n%s", want, out)
		}
	}
}

func TestSetupCmd_DryRunPlan(t *testing.T) {
	out, err := run(t, "setup", "--chain", "wbft", "--validators", "3", "--endpoints", "1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "chain_id 8284") {
		t.Errorf("expected wbft chain_id 8284:\n%s", out)
	}
	// 3 validators + 1 endpoint = 4 node rows; check node 4 present.
	if !strings.Contains(out, "\n4\t") && !strings.Contains(out, "4  ") {
		t.Errorf("expected 4 nodes in plan:\n%s", out)
	}
	if !strings.Contains(out, "genesis:  template=true") {
		t.Errorf("wbft should report a genesis template:\n%s", out)
	}
}

func TestSetupCmd_LaunchNotWired(t *testing.T) {
	_, err := run(t, "setup", "--chain", "stablenet", "--dry-run=false")
	if err == nil || !strings.Contains(err.Error(), "not yet wired") {
		t.Errorf("expected not-yet-wired error, got %v", err)
	}
}

func TestVerifyCmd_HTTPMock(t *testing.T) {
	var block int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		switch req.Method {
		case "eth_blockNumber":
			block++
			res["result"] = "0x" + strings.TrimPrefix(itoaHex(block), "0x")
		case "eth_chainId":
			res["result"] = "0x205b"
		case "net_peerCount":
			res["result"] = "0x2"
		case "eth_syncing":
			res["result"] = false
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	defer srv.Close()

	out, err := run(t, "verify", "--rpc", srv.URL, "--progress-delay", "1ms")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "producing: true") {
		t.Errorf("expected producing:true:\n%s", out)
	}
	if !strings.Contains(out, "8283") {
		t.Errorf("expected chain id 8283:\n%s", out)
	}
}

func TestFaucetCmd_HTTPMock(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		switch req.Method {
		case "eth_chainId":
			res["result"] = "0x205b"
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
	defer srv.Close()

	out, err := run(t, "faucet",
		"--chain", "stablenet",
		"--rpc", srv.URL,
		"--from-key", "0000000000000000000000000000000000000000000000000000000000000001",
		"--to", "0x7e5f4552091a69125d5dfcb7b8c2659029395bdf",
		"--amount", "1000000",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "0x") {
		t.Errorf("expected tx hash, got %q", out)
	}
}

// itoaHex renders a small int as a hex quantity string.
func itoaHex(n int) string {
	const digits = "0123456789abcdef"
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{digits[n%16]}, b...)
		n /= 16
	}
	return string(b)
}
