package remote

import (
	"context"
	"encoding/json"
	"errors"
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

// TestClient_NonceAt_Happy asserts the NonceAt wrapper forwards
// eth_getTransactionCount at the given block tag and returns the parsed nonce.
// Distinct from PendingNonceAt: NonceAt reads the historical/latest mined
// nonce rather than including pending-pool entries.
func TestClient_NonceAt_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getTransactionCount" {
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
		t.Fatal(err)
	}
	defer c.Close()

	n, err := c.NonceAt(ctx, common.HexToAddress("0x01"), nil)
	if err != nil {
		t.Fatalf("NonceAt: %v", err)
	}
	if n != 42 {
		t.Errorf("nonce = %d, want 42", n)
	}
}

// TestClient_NonceAt_Reject asserts the wrapper-prefix on RPC error so
// callers can distinguish remote vs local failures by the error string.
func TestClient_NonceAt_Reject(t *testing.T) {
	srv := mockRPC(t, map[string]string{})
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, err = c.NonceAt(ctx, common.HexToAddress("0x01"), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "remote.NonceAt") {
		t.Errorf("err missing wrap prefix: %v", err)
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

func TestClient_TransactionReceipt_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getTransactionReceipt" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{
                "transactionHash":"0x%s",
                "blockHash":"0x2222222222222222222222222222222222222222222222222222222222222222",
                "blockNumber":"0x1",
                "cumulativeGasUsed":"0x5208",
                "gasUsed":"0x5208",
                "status":"0x1",
                "contractAddress":null,
                "logsBloom":"0x%s",
                "logs":[]}}`, req.ID, strings.Repeat("a", 64), strings.Repeat("0", 512))
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
	h := common.HexToHash("0x" + strings.Repeat("a", 64))
	rcpt, err := c.TransactionReceipt(ctx, h)
	if err != nil {
		t.Fatalf("TransactionReceipt: %v", err)
	}
	if rcpt.Status != 1 {
		t.Errorf("status = %d, want 1", rcpt.Status)
	}
}

func TestClient_TransactionReceipt_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getTransactionReceipt" {
			// ethclient.TransactionReceipt returns ethereum.NotFound when result is null.
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":null}`, req.ID)
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
	_, err = c.TransactionReceipt(ctx, common.HexToHash("0x"+strings.Repeat("b", 64)))
	if !errors.Is(err, ethereum.NotFound) {
		t.Errorf("err = %v, want ethereum.NotFound", err)
	}
}

// TestClient_SendRawTransaction_Happy asserts the wrapper hex-encodes the
// raw bytes once and forwards them as the single string param to
// eth_sendRawTransaction. The mock captures Params[0] and verifies it is
// the canonical lowercase hex of the input ([]byte{0x16, 0xc0} -> "0x16c0").
func TestClient_SendRawTransaction_Happy(t *testing.T) {
	var sentParam string
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
				_ = json.Unmarshal(req.Params[0], &sentParam)
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x%s"}`, req.ID, strings.Repeat("a", 64))
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
	raw := []byte{0x16, 0xc0}
	if err := c.SendRawTransaction(ctx, raw); err != nil {
		t.Fatalf("SendRawTransaction: %v", err)
	}
	if sentParam != "0x16c0" {
		t.Errorf("sent param = %q, want 0x16c0", sentParam)
	}
}

// TestClient_SendRawTransaction_Reject asserts an endpoint rejection
// (RPC error -32000) is surfaced through the remote.SendRawTransaction
// wrapper prefix so callers can recognize broadcast failures without
// string-matching the upstream message.
func TestClient_SendRawTransaction_Reject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"invalid tx"}}`, req.ID)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	err = c.SendRawTransaction(ctx, []byte{0x16, 0xc0})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "remote.SendRawTransaction") {
		t.Errorf("err missing wrap prefix: %v", err)
	}
}

// TestClient_CallContract_Happy asserts the wrapper forwards eth_call and
// returns the hex-decoded result bytes. The mock recognizes eth_call and
// echoes back a simple ABI-encoded uint256 value (0x2a -> 42).
func TestClient_CallContract_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_call" {
			// 32-byte big-endian encoding of 42.
			result := "0x" + strings.Repeat("0", 62) + "2a"
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, result)
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
	to := common.HexToAddress("0x01")
	out, err := c.CallContract(ctx, ethereum.CallMsg{To: &to, Data: []byte{0x12, 0x34}}, nil)
	if err != nil {
		t.Fatalf("CallContract: %v", err)
	}
	if len(out) != 32 || out[31] != 0x2a {
		t.Errorf("CallContract result = %x, want 32-byte 42", out)
	}
}

// TestClient_CallContract_Reject asserts an endpoint rejection surfaces
// through the remote.CallContract wrapper prefix.
func TestClient_CallContract_Reject(t *testing.T) {
	srv := mockRPC(t, map[string]string{}) // no methods -> fail
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	to := common.HexToAddress("0x01")
	_, err = c.CallContract(ctx, ethereum.CallMsg{To: &to}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "remote.CallContract") {
		t.Errorf("err missing wrap prefix: %v", err)
	}
}

// TestClient_FilterLogs_Happy asserts eth_getLogs is forwarded and a
// minimal log result is parsed back into types.Log.
func TestClient_FilterLogs_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getLogs" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":[{
                "address":"0x0000000000000000000000000000000000000001",
                "topics":["0x%s"],
                "data":"0x",
                "blockNumber":"0x1",
                "transactionHash":"0x%s",
                "transactionIndex":"0x0",
                "blockHash":"0x%s",
                "logIndex":"0x0",
                "removed":false}]}`, req.ID,
				strings.Repeat("a", 64), strings.Repeat("b", 64), strings.Repeat("c", 64))
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
	logs, err := c.FilterLogs(ctx, ethereum.FilterQuery{})
	if err != nil {
		t.Fatalf("FilterLogs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(logs))
	}
	if logs[0].Address != common.HexToAddress("0x01") {
		t.Errorf("log addr = %s, want 0x01", logs[0].Address.Hex())
	}
}

// TestClient_FilterLogs_Reject asserts the wrapper prefix surfaces on RPC error.
func TestClient_FilterLogs_Reject(t *testing.T) {
	srv := mockRPC(t, map[string]string{})
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, err = c.FilterLogs(ctx, ethereum.FilterQuery{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "remote.FilterLogs") {
		t.Errorf("err missing wrap prefix: %v", err)
	}
}

// TestClient_CodeAt_Happy asserts eth_getCode is forwarded and bytes return.
func TestClient_CodeAt_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getCode" {
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0xdeadbeef"}`, req.ID)
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
	code, err := c.CodeAt(ctx, common.HexToAddress("0x01"), nil)
	if err != nil {
		t.Fatalf("CodeAt: %v", err)
	}
	want := []byte{0xde, 0xad, 0xbe, 0xef}
	if len(code) != 4 || code[0] != want[0] || code[3] != want[3] {
		t.Errorf("code = %x, want %x", code, want)
	}
}

// TestClient_CodeAt_Reject asserts wrapper prefix on RPC error.
func TestClient_CodeAt_Reject(t *testing.T) {
	srv := mockRPC(t, map[string]string{})
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, err = c.CodeAt(ctx, common.HexToAddress("0x01"), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "remote.CodeAt") {
		t.Errorf("err missing wrap prefix: %v", err)
	}
}

// TestClient_StorageAt_Happy asserts eth_getStorageAt is forwarded and the
// 32-byte word is parsed back.
func TestClient_StorageAt_Happy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string
			ID     json.RawMessage
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		if req.Method == "eth_getStorageAt" {
			result := "0x" + strings.Repeat("0", 62) + "ff"
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%q}`, req.ID, result)
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
	val, err := c.StorageAt(ctx, common.HexToAddress("0x01"), common.Hash{}, nil)
	if err != nil {
		t.Fatalf("StorageAt: %v", err)
	}
	if len(val) != 32 || val[31] != 0xff {
		t.Errorf("storage = %x, want last byte 0xff", val)
	}
}

// TestClient_StorageAt_Reject asserts wrapper prefix on RPC error.
func TestClient_StorageAt_Reject(t *testing.T) {
	srv := mockRPC(t, map[string]string{})
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	c, err := Dial(ctx, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, err = c.StorageAt(ctx, common.HexToAddress("0x01"), common.Hash{}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "remote.StorageAt") {
		t.Errorf("err missing wrap prefix: %v", err)
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
