package probe

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type rpcRequest struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
	ID     int    `json:"id"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	Result any       `json:"result,omitempty"`
	Error  *rpcError `json:"error,omitempty"`
	ID     int       `json:"id"`
}

// mockRPC returns an httptest.Server that dispatches by method name.
// handlers[method] returns (result, errCode). errCode 0 = success (result only).
// Missing method -> -32601 method-not-found.
func mockRPC(t *testing.T, handlers map[string]func() (any, int)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req rpcRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		resp := rpcResponse{ID: req.ID}
		if h, ok := handlers[req.Method]; ok {
			result, code := h()
			if code == 0 {
				resp.Result = result
			} else {
				resp.Error = &rpcError{Code: code, Message: "mock error"}
			}
		} else {
			resp.Error = &rpcError{Code: -32601, Message: "method not found"}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestDetect(t *testing.T) {
	cases := []struct {
		name           string
		handlers       map[string]func() (any, int)
		override       string
		wantChainType  string
		wantChainID    int64
		wantNamespaces []string
		wantOverridden bool
	}{
		{
			name: "stablenet via istanbul + chain_id 8283",
			handlers: map[string]func() (any, int){
				"eth_chainId":            func() (any, int) { return "0x205b", 0 },
				"istanbul_getValidators": func() (any, int) { return []string{}, 0 },
			},
			wantChainType:  "stablenet",
			wantChainID:    8283,
			wantNamespaces: []string{"istanbul"},
		},
		{
			name: "wbft via istanbul + non-stablenet chain_id",
			handlers: map[string]func() (any, int){
				"eth_chainId":            func() (any, int) { return "0x7a69", 0 }, // 31337
				"istanbul_getValidators": func() (any, int) { return []string{}, 0 },
			},
			wantChainType:  "wbft",
			wantChainID:    31337,
			wantNamespaces: []string{"istanbul"},
		},
		{
			name: "wemix via wemix namespace",
			handlers: map[string]func() (any, int){
				"eth_chainId":     func() (any, int) { return "0x3e9", 0 }, // 1001
				"wemix_getReward": func() (any, int) { return "0x0", 0 },
			},
			wantChainType:  "wemix",
			wantChainID:    1001,
			wantNamespaces: []string{"wemix"},
		},
		{
			name: "ethereum fallback",
			handlers: map[string]func() (any, int){
				"eth_chainId": func() (any, int) { return "0x1", 0 },
			},
			wantChainType:  "ethereum",
			wantChainID:    1,
			wantNamespaces: []string{},
		},
		{
			name: "override stablenet short-circuits",
			handlers: map[string]func() (any, int){
				"eth_chainId": func() (any, int) { return "0x1", 0 }, // mismatched id
			},
			override:       "stablenet",
			wantChainType:  "stablenet",
			wantChainID:    1,
			wantNamespaces: []string{},
			wantOverridden: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := mockRPC(t, tc.handlers)
			defer srv.Close()
			res, err := Detect(context.Background(), Options{
				RPCURL:   srv.URL,
				Timeout:  2 * time.Second,
				Override: tc.override,
			})
			if err != nil {
				t.Fatalf("Detect: %v", err)
			}
			if res.ChainType != tc.wantChainType {
				t.Errorf("ChainType = %q, want %q", res.ChainType, tc.wantChainType)
			}
			if res.ChainID != tc.wantChainID {
				t.Errorf("ChainID = %d, want %d", res.ChainID, tc.wantChainID)
			}
			if res.Overridden != tc.wantOverridden {
				t.Errorf("Overridden = %v, want %v", res.Overridden, tc.wantOverridden)
			}
			if len(res.Namespaces) != len(tc.wantNamespaces) {
				t.Errorf("Namespaces = %v, want %v", res.Namespaces, tc.wantNamespaces)
			} else {
				for i, ns := range tc.wantNamespaces {
					if res.Namespaces[i] != ns {
						t.Errorf("Namespaces[%d] = %q, want %q", i, res.Namespaces[i], ns)
					}
				}
			}
		})
	}
}

func TestDetect_EthChainIDFails(t *testing.T) {
	srv := mockRPC(t, map[string]func() (any, int){
		"eth_chainId": func() (any, int) { return nil, -32000 },
	})
	defer srv.Close()
	_, err := Detect(context.Background(), Options{RPCURL: srv.URL, Timeout: time.Second})
	if err == nil {
		t.Fatal("expected error for eth_chainId RPC error")
	}
}

func TestDetect_RejectsNonHTTP(t *testing.T) {
	_, err := Detect(context.Background(), Options{RPCURL: "ws://x", Timeout: time.Second})
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatalf("err = %v, want wrapped ErrInvalidURL", err)
	}
	if !IsInputError(err) {
		t.Error("IsInputError should classify non-http scheme as input error")
	}
}

func TestDetect_UnknownOverride(t *testing.T) {
	srv := mockRPC(t, map[string]func() (any, int){
		"eth_chainId": func() (any, int) { return "0x1", 0 },
	})
	defer srv.Close()
	_, err := Detect(context.Background(), Options{RPCURL: srv.URL, Override: "fakechain"})
	if !errors.Is(err, ErrUnknownOverride) {
		t.Fatalf("err = %v, want wrapped ErrUnknownOverride", err)
	}
	if !IsInputError(err) {
		t.Error("IsInputError should classify unknown override as input error")
	}
}

func TestDetect_MissingURL(t *testing.T) {
	_, err := Detect(context.Background(), Options{})
	if !errors.Is(err, ErrMissingURL) {
		t.Fatalf("err = %v, want ErrMissingURL", err)
	}
	if !IsInputError(err) {
		t.Error("IsInputError should classify missing URL as input error")
	}
}

func TestIsInputError_UpstreamNotClassifiedAsInput(t *testing.T) {
	srv := mockRPC(t, map[string]func() (any, int){
		"eth_chainId": func() (any, int) { return nil, -32000 },
	})
	defer srv.Close()
	_, err := Detect(context.Background(), Options{RPCURL: srv.URL, Timeout: time.Second})
	if err == nil {
		t.Fatal("expected upstream error")
	}
	if IsInputError(err) {
		t.Error("IsInputError should NOT classify upstream RPC error as input error")
	}
}

func TestDetect_ChainIDNotAString(t *testing.T) {
	srv := mockRPC(t, map[string]func() (any, int){
		"eth_chainId": func() (any, int) { return 12345, 0 }, // number, not hex string
	})
	defer srv.Close()
	_, err := Detect(context.Background(), Options{RPCURL: srv.URL, Timeout: time.Second})
	if err == nil {
		t.Fatal("expected error for non-string chainId")
	}
}

func TestDetect_ChainIDBadHex(t *testing.T) {
	srv := mockRPC(t, map[string]func() (any, int){
		"eth_chainId": func() (any, int) { return "0xZZZ", 0 },
	})
	defer srv.Close()
	_, err := Detect(context.Background(), Options{RPCURL: srv.URL, Timeout: time.Second})
	if err == nil {
		t.Fatal("expected error for unparseable chainId")
	}
}

func TestDetect_HTTPNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	_, err := Detect(context.Background(), Options{RPCURL: srv.URL, Timeout: time.Second})
	if err == nil {
		t.Fatal("expected error for non-200 HTTP")
	}
}

func TestDetect_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()
	_, err := Detect(context.Background(), Options{RPCURL: srv.URL, Timeout: time.Second})
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestDetect_NetworkError(t *testing.T) {
	// Port 1 on localhost is reliably unreachable/refused.
	_, err := Detect(context.Background(), Options{RPCURL: "http://127.0.0.1:1", Timeout: 200 * time.Millisecond})
	if err == nil {
		t.Fatal("expected error for unreachable endpoint")
	}
}

func TestDetect_OversizedBodyCapped(t *testing.T) {
	// Hostile endpoint streams a body larger than maxRPCResponseBytes.
	// LimitReader should cap the read; decode then fails cleanly as
	// truncated JSON rather than consuming unbounded memory.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"`))
		// Write enough to exceed the 1 MiB cap; truncation during decode proves the bound.
		chunk := strings.Repeat("A", 4096)
		for range 300 {
			if _, err := w.Write([]byte(chunk)); err != nil {
				return
			}
		}
		_, _ = w.Write([]byte(`"}`))
	}))
	defer srv.Close()
	_, err := Detect(context.Background(), Options{RPCURL: srv.URL, Timeout: 2 * time.Second})
	if err == nil {
		t.Fatal("expected error: oversized body should cause decode failure after cap")
	}
}

func TestProbeMethod_RPCErrorReturnsFalse(t *testing.T) {
	srv := mockRPC(t, map[string]func() (any, int){
		"eth_chainId":            func() (any, int) { return "0x1", 0 },
		"istanbul_getValidators": func() (any, int) { return nil, -32602 }, // invalid params
	})
	defer srv.Close()
	// istanbul probe returns RPC error -> probeMethod should treat as unavailable ->
	// classification must fall through to ethereum, not misclassify as wbft.
	res, err := Detect(context.Background(), Options{RPCURL: srv.URL, Timeout: time.Second})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if res.ChainType != "ethereum" {
		t.Errorf("ChainType = %q, want ethereum (RPC error on probe should mark unavailable)", res.ChainType)
	}
}
