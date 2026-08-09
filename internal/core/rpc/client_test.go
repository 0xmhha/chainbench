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

func TestClient_HeadBlock(t *testing.T) {
	srv := rpcServer(t, map[string]any{
		"eth_getBlockByNumber": map[string]any{
			"number": "0x10", // 16
			"hash":   "0xabc",
			"miner":  "0xVAL1",
		},
	})
	defer srv.Close()
	c := Dial(srv.URL)

	num, hash, miner, err := c.HeadBlock(context.Background())
	if err != nil {
		t.Fatalf("HeadBlock: %v", err)
	}
	if num != 16 || hash != "0xabc" || miner != "0xVAL1" {
		t.Fatalf("HeadBlock = (%d, %q, %q), want (16, 0xabc, 0xVAL1)", num, hash, miner)
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

func TestClient_Coinbase(t *testing.T) {
	srv := rpcServer(t, map[string]any{"eth_coinbase": "0xc17d493883eaa3b4cceb0f214b273392d562f9d8"})
	defer srv.Close()
	cb, err := Dial(srv.URL).Coinbase(context.Background())
	if err != nil || cb != "0xc17d493883eaa3b4cceb0f214b273392d562f9d8" {
		t.Errorf("Coinbase: %q, %v", cb, err)
	}
}

func TestClient_SendTransaction(t *testing.T) {
	// Capture the params so the node-side tx args (from/to/data) are asserted to
	// travel, and return a tx hash.
	var got struct {
		Method string          `json:"method"`
		Params []SendTxArgs    `json:"params"`
		ID     json.RawMessage `json:"id"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": 1, "result": "0xdeadbeef",
		})
	}))
	defer srv.Close()

	hash, err := Dial(srv.URL).SendTransaction(context.Background(), SendTxArgs{
		From: "0xfrom", To: "0xto", Data: "0xabcd", Gas: "0x1e8480",
	})
	if err != nil || hash != "0xdeadbeef" {
		t.Fatalf("SendTransaction: %q, %v", hash, err)
	}
	if got.Method != "eth_sendTransaction" || len(got.Params) != 1 {
		t.Fatalf("bad request: %+v", got)
	}
	if a := got.Params[0]; a.From != "0xfrom" || a.To != "0xto" || a.Data != "0xabcd" || a.Gas != "0x1e8480" {
		t.Errorf("tx args did not travel: %+v", a)
	}
}

func TestClient_BlockByNumber(t *testing.T) {
	srv := rpcServer(t, map[string]any{
		"eth_getBlockByNumber": map[string]any{
			"number": "0x0", "hash": "0xgenesis", "miner": "0x0", "timestamp": "0x64",
		},
	})
	defer srv.Close()
	c := Dial(srv.URL)

	b, err := c.BlockByNumber(context.Background(), "0x0")
	if err != nil {
		t.Fatalf("BlockByNumber: %v", err)
	}
	if b.Number != 0 || b.Hash != "0xgenesis" || b.Timestamp != 100 {
		t.Fatalf("block = %+v, want {0, 0xgenesis, ts 100}", b)
	}
}

func TestClient_BlockByNumber_BaseFee(t *testing.T) {
	srv := rpcServer(t, map[string]any{
		"eth_getBlockByNumber": map[string]any{
			"number": "0x1", "hash": "0xh", "baseFeePerGas": "0x12309ce54000", // 20000000000000
		},
	})
	defer srv.Close()
	b, err := Dial(srv.URL).BlockByNumber(context.Background(), "latest")
	if err != nil {
		t.Fatalf("BlockByNumber: %v", err)
	}
	if b.BaseFeePerGas == nil || b.BaseFeePerGas.String() != "20000000000000" {
		t.Fatalf("baseFeePerGas = %v, want 20000000000000", b.BaseFeePerGas)
	}
}

func TestClient_EstimateGas(t *testing.T) {
	srv := rpcServer(t, map[string]any{"eth_estimateGas": "0x8000"}) // 32768
	defer srv.Close()
	g, err := Dial(srv.URL).EstimateGas(context.Background(), "0xfrom", "0xto", "0xa9059cbb")
	if err != nil || g != 32768 {
		t.Fatalf("EstimateGas: %d, %v", g, err)
	}
}
