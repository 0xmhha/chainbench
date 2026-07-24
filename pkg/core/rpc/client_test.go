package rpc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func rpcServer(t *testing.T, results map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		v, ok := results[req.Method]
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if !ok {
			resp["error"] = map[string]any{"code": -32601, "message": "method not found"}
		} else {
			resp["result"] = v
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestClient_Quantities(t *testing.T) {
	srv := rpcServer(t, map[string]any{
		"eth_blockNumber": "0x2a",   // 42
		"eth_chainId":     "0x205b", // 8283
		"net_peerCount":   "0x3",
		"eth_syncing":     false,
	})
	defer srv.Close()
	c := Dial(srv.URL)
	ctx := context.Background()

	if bn, err := c.BlockNumber(ctx); err != nil || bn != 42 {
		t.Errorf("BlockNumber: %d, %v", bn, err)
	}
	if id, err := c.ChainID(ctx); err != nil || id != 8283 {
		t.Errorf("ChainID: %d, %v", id, err)
	}
	if pc, err := c.PeerCount(ctx); err != nil || pc != 3 {
		t.Errorf("PeerCount: %d, %v", pc, err)
	}
	if syncing, err := c.Syncing(ctx); err != nil || syncing {
		t.Errorf("Syncing: %v, %v", syncing, err)
	}
}

func TestClient_AccountReads(t *testing.T) {
	srv := rpcServer(t, map[string]any{
		"eth_getBalance":          "0xde0b6b3a7640000", // 1e18
		"eth_getTransactionCount": "0x5",
		"eth_getCode":             "0x6001",
		"eth_call":                "0x2a",
	})
	defer srv.Close()
	c := Dial(srv.URL)
	ctx := context.Background()

	if bal, err := c.BalanceAt(ctx, "0xabc"); err != nil || bal.String() != "1000000000000000000" {
		t.Errorf("BalanceAt: %v, %v", bal, err)
	}
	if n, err := c.NonceAt(ctx, "0xabc"); err != nil || n != 5 {
		t.Errorf("NonceAt: %d, %v", n, err)
	}
	if code, err := c.CodeAt(ctx, "0xabc"); err != nil || code != "0x6001" {
		t.Errorf("CodeAt: %q, %v", code, err)
	}
	if res, err := c.EthCall(ctx, "0xdef", "0x00"); err != nil || res != "0x2a" {
		t.Errorf("EthCall: %q, %v", res, err)
	}
}

func TestClient_TxReceipt(t *testing.T) {
	ctx := context.Background()

	pending := rpcServer(t, map[string]any{"eth_getTransactionReceipt": nil})
	defer pending.Close()
	if r, err := Dial(pending.URL).TxReceipt(ctx, "0xhash"); err != nil || r != nil {
		t.Errorf("pending receipt should be nil: %s, %v", r, err)
	}

	mined := rpcServer(t, map[string]any{
		"eth_getTransactionReceipt": map[string]any{"status": "0x1", "blockNumber": "0x5"},
	})
	defer mined.Close()
	r, err := Dial(mined.URL).TxReceipt(ctx, "0xhash")
	if err != nil || r == nil || !strings.Contains(string(r), "0x5") {
		t.Errorf("mined receipt: %s, %v", r, err)
	}
}

func TestClient_SyncingObjectMeansSyncing(t *testing.T) {
	srv := rpcServer(t, map[string]any{
		"eth_syncing": map[string]any{"currentBlock": "0x1", "highestBlock": "0x9"},
	})
	defer srv.Close()
	if s, err := Dial(srv.URL).Syncing(context.Background()); err != nil || !s {
		t.Errorf("expected syncing=true, got %v (%v)", s, err)
	}
}

func TestClient_ServerError(t *testing.T) {
	srv := rpcServer(t, map[string]any{}) // every method -> error
	defer srv.Close()
	if _, err := Dial(srv.URL).BlockNumber(context.Background()); err == nil {
		t.Error("expected server error")
	}
}
