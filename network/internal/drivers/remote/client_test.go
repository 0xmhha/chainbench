package remote

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestClient_DialRejectsBadURL(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := Dial(ctx, "not-a-url")
	if err == nil {
		t.Fatal("expected Dial error for malformed URL")
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
