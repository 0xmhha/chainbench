package remote

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
