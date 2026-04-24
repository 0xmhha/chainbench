package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// mockRPC returns a JSON-RPC server responding to eth_blockNumber with the given hex.
// Unknown methods return -32601.
func mockRPC(t *testing.T, responses map[string]string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if result, ok := responses[req.Method]; ok {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"` + result + `"}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"method not found"}}`))
	}))
}

func TestClient_BlockNumber(t *testing.T) {
	srv := mockRPC(t, map[string]string{"eth_blockNumber": "0x10"})
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	bn, err := c.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("BlockNumber: %v", err)
	}
	if bn != 16 {
		t.Errorf("BlockNumber = %d, want 16", bn)
	}
}

// "not-a-url" surfaces through ethclient as an unknown-transport-scheme
// failure (url.Parse accepts it; rpc.DialContext rejects the empty scheme).
// We assert the error flows through our Dial wrapper so callers see the
// endpoint URL in the message.
func TestClient_DialRejectsUnsupportedScheme(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := Dial(ctx, "not-a-url")
	if err == nil {
		t.Fatal("expected Dial error for unsupported URL scheme")
	}
	if !strings.Contains(err.Error(), "remote.Dial") {
		t.Errorf("err should flow through remote.Dial wrapper: %v", err)
	}
}

// Regression-proofs the ctx.Done() contract: a slow RPC must surface as an
// error on ctx deadline rather than blocking the caller past it. Uses a
// bounded handler sleep (not an indefinite hang) to avoid test teardown
// stalling on a keep-alive connection that the client already abandoned.
func TestClient_BlockNumber_HonorsContextTimeout(t *testing.T) {
	done := make(chan struct{})
	defer close(done)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(2 * time.Second):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x0"}`))
		case <-done:
		}
	}))
	defer srv.Close()

	dialCtx, dialCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer dialCancel()
	c, err := Dial(dialCtx, srv.URL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	callCtx, callCancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer callCancel()
	start := time.Now()
	_, err = c.BlockNumber(callCtx)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error when context deadline passes during RPC")
	}
	if elapsed > time.Second {
		t.Errorf("BlockNumber blocked %v past the 150ms deadline", elapsed)
	}
}

func TestClient_BlockNumber_RPCError(t *testing.T) {
	srv := mockRPC(t, map[string]string{}) // no methods — every call fails
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()
	_, err = c.BlockNumber(ctx)
	if err == nil {
		t.Fatal("expected error when server returns method-not-found")
	}
}

// Verifies that a custom RoundTripper passed via DialOptions.Transport is
// actually applied to outbound RPC traffic (the ethclient round-trip goes
// through our injected transport).
func TestDialWithOptions_TransportInjected(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("X-Test-Auth")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x5"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rt := APIKeyTransport(nil, "X-Test-Auth", "mykey")
	c, err := DialWithOptions(ctx, srv.URL, DialOptions{Transport: rt})
	if err != nil {
		t.Fatalf("DialWithOptions: %v", err)
	}
	defer c.Close()

	bn, err := c.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("BlockNumber: %v", err)
	}
	if bn != 5 {
		t.Errorf("bn = %d, want 5", bn)
	}
	if gotAuth != "mykey" {
		t.Errorf("header not injected: %q", gotAuth)
	}
}

func TestClient_ChainID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_chainId" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x2a"}`, req.ID)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()
	cid, err := c.ChainID(ctx)
	if err != nil {
		t.Fatalf("ChainID: %v", err)
	}
	if cid.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("ChainID = %v, want 42", cid)
	}
}

func TestClient_BalanceAt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getBalance" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x100"}`, req.ID)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()
	addr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	bal, err := c.BalanceAt(ctx, addr, nil) // latest
	if err != nil {
		t.Fatalf("BalanceAt: %v", err)
	}
	if bal.Cmp(big.NewInt(0x100)) != 0 {
		t.Errorf("balance = %v, want 0x100", bal)
	}
}

func TestClient_GasPrice(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_gasPrice" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x3b9aca00"}`, req.ID) // 1 gwei
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()
	gp, err := c.GasPrice(ctx)
	if err != nil {
		t.Fatalf("GasPrice: %v", err)
	}
	if gp.Cmp(big.NewInt(1_000_000_000)) != 0 {
		t.Errorf("gas_price = %v, want 1 gwei", gp)
	}
}

// Error-branch coverage for the three new wrappers: each must surface the
// remote.<Method> wrapper prefix when the underlying JSON-RPC call returns
// method-not-found. Parallels TestClient_BlockNumber_RPCError.
func TestClient_ChainID_RPCError(t *testing.T) {
	srv := mockRPC(t, map[string]string{}) // no methods — every call fails
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()
	if _, err := c.ChainID(ctx); err == nil {
		t.Fatal("expected error when server returns method-not-found")
	} else if !strings.Contains(err.Error(), "remote.ChainID") {
		t.Errorf("err should flow through remote.ChainID wrapper: %v", err)
	}
}

func TestClient_BalanceAt_RPCError(t *testing.T) {
	srv := mockRPC(t, map[string]string{})
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()
	addr := common.HexToAddress("0x0000000000000000000000000000000000000001")
	if _, err := c.BalanceAt(ctx, addr, nil); err == nil {
		t.Fatal("expected error when server returns method-not-found")
	} else if !strings.Contains(err.Error(), "remote.BalanceAt") {
		t.Errorf("err should flow through remote.BalanceAt wrapper: %v", err)
	}
}

func TestClient_GasPrice_RPCError(t *testing.T) {
	srv := mockRPC(t, map[string]string{})
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()
	if _, err := c.GasPrice(ctx); err == nil {
		t.Fatal("expected error when server returns method-not-found")
	} else if !strings.Contains(err.Error(), "remote.GasPrice") {
		t.Errorf("err should flow through remote.GasPrice wrapper: %v", err)
	}
}

// TestClient_PendingNonceAt asserts the PendingNonceAt wrapper forwards
// eth_getTransactionCount with the pending block tag and returns the parsed
// nonce. The mock only needs to recognize the method name — ethclient
// supplies the "pending" tag internally.
func TestClient_PendingNonceAt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getTransactionCount" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x7"}`, req.ID)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	n, err := c.PendingNonceAt(ctx, common.HexToAddress("0x01"))
	if err != nil {
		t.Fatalf("PendingNonceAt: %v", err)
	}
	if n != 7 {
		t.Errorf("nonce = %d, want 7", n)
	}
}

// TestClient_EstimateGas asserts eth_estimateGas is forwarded and the hex
// result is parsed back to uint64 gas units.
func TestClient_EstimateGas(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_estimateGas" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x5208"}`, req.ID) // 21000
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	from := common.HexToAddress("0x01")
	to := common.HexToAddress("0x02")
	gas, err := c.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &to})
	if err != nil {
		t.Fatalf("EstimateGas: %v", err)
	}
	if gas != 21000 {
		t.Errorf("gas = %d, want 21000", gas)
	}
}

// TestClient_SendTransaction verifies the wrapper RLP-encodes and forwards a
// tx via eth_sendRawTransaction. The mock records Params[0] (the hex RLP)
// and asserts it's non-empty; signature validity is not inspected because
// ethclient will happily serialize an unsigned LegacyTx.
func TestClient_SendTransaction(t *testing.T) {
	var receivedHex string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
			Params []json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_sendRawTransaction" {
			if len(req.Params) > 0 {
				_ = json.Unmarshal(req.Params[0], &receivedHex)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0xabc123"}`, req.ID)
			return
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	tx := types.NewTx(&types.LegacyTx{Nonce: 0, GasPrice: big.NewInt(1), Gas: 21000})
	if err := c.SendTransaction(ctx, tx); err != nil {
		t.Fatalf("SendTransaction: %v", err)
	}
	if receivedHex == "" {
		t.Error("server did not receive a signed tx param")
	}
}

// Guards against the silent-bypass hazard: passing a Transport for a non-HTTP
// URL (ws/wss/ipc) must error loudly since WithHTTPClient is ignored by
// those schemes and auth headers would be dropped.
func TestDialWithOptions_RejectsNonHTTPWithTransport(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	rt := APIKeyTransport(nil, "X-Test-Auth", "mykey")

	cases := []string{
		"ws://localhost:8546",
		"wss://localhost:8546",
		"/tmp/geth.ipc",
	}
	for _, url := range cases {
		t.Run(url, func(t *testing.T) {
			_, err := DialWithOptions(ctx, url, DialOptions{Transport: rt})
			if err == nil {
				t.Fatalf("expected error for non-HTTP URL %q with Transport", url)
			}
			if !strings.Contains(err.Error(), "http(s)") {
				t.Errorf("error should mention http(s) requirement: %v", err)
			}
		})
	}
}
