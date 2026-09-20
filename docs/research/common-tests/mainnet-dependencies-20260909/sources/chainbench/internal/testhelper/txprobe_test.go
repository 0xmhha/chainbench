package testhelper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"encoding/json"
	"io"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// mockReceipts serves eth_getTransactionReceipt: a hash in mined gets a receipt,
// any other hash gets null (not yet on chain).
func mockReceipts(t *testing.T, mined map[string]bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		_ = json.Unmarshal(body, &req)
		var result any
		if req.Method == "eth_getTransactionReceipt" {
			hash, _ := req.Params[0].(string)
			if mined[hash] {
				result = map[string]any{"status": "0x1", "transactionHash": hash}
			} else {
				result = nil
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestTxMined_ReaderAndAssertion(t *testing.T) {
	srv := mockReceipts(t, map[string]bool{"0xmined": true})
	d := deps()
	on := []node.Node{{Index: 1, RPCURL: srv.URL}}

	as, ok := d.Actions.Assertion(assertTxMined)
	if !ok {
		t.Fatal("txMined assertion not registered")
	}
	cases := []struct {
		hash     string
		expected string
		pass     bool
	}{
		{"0xmined", "true", true},
		{"0xmined", "false", false},
		{"0xpending", "false", true},
		{"0xpending", "true", false},
	}
	for _, tc := range cases {
		r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on,
			Spec: map[string]any{"assert": assertTxMined, "hash": tc.hash, "expected": tc.expected}})
		if err != nil {
			t.Fatalf("txMined %s: %v", tc.hash, err)
		}
		if r.Pass != tc.pass {
			t.Errorf("txMined(%s)==%s Pass = %v, want %v", tc.hash, tc.expected, r.Pass, tc.pass)
		}
	}

	// The same name is a read source, for waitFor.
	if _, ok := d.Actions.Reader(assertTxMined); !ok {
		t.Error("txMined must also be a read source")
	}
}

func TestCallError_PassesOnRevert(t *testing.T) {
	// A server that errors eth_call (a revert) and one that answers it.
	reverting := mockRPCReject(t, "execution reverted: BAD_INPUT")
	answering := mockRPC(t, map[string]any{"eth_call": "0x01"})
	d := deps()
	as, _ := d.Actions.Assertion(assertCallError)

	spec := map[string]any{"assert": assertCallError, "to": "0xabc", "data": "0xa9cc4718"}
	r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d,
		On: []node.Node{{Index: 1, RPCURL: reverting.URL}}, Spec: spec})
	if err != nil || !r.Pass {
		t.Fatalf("a reverting call must pass callError: pass=%v err=%v", r.Pass, err)
	}
	r, err = as.Check(context.Background(), &interp.AssertCtx{Deps: &d,
		On: []node.Node{{Index: 1, RPCURL: answering.URL}}, Spec: spec})
	if err != nil {
		t.Fatalf("callError check: %v", err)
	}
	if r.Pass {
		t.Error("a call that returns a value must fail callError")
	}
}

func TestMethodPresent(t *testing.T) {
	present := mockRPCReject(t, "invalid argument 0") // registered, rejects the probe
	absent := mockRPCReject(t, "the method x does not exist/is not available")
	d := deps()
	as, _ := d.Actions.Assertion(assertMethodPresent)
	spec := map[string]any{"assert": assertMethodPresent, "method": "eth_signRawFeeDelegateTransaction"}

	r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d,
		On: []node.Node{{Index: 1, RPCURL: present.URL}}, Spec: spec})
	if err != nil || !r.Pass {
		t.Fatalf("a registered method (non-not-found error) must pass: pass=%v err=%v", r.Pass, err)
	}
	r, _ = as.Check(context.Background(), &interp.AssertCtx{Deps: &d,
		On: []node.Node{{Index: 1, RPCURL: absent.URL}}, Spec: spec})
	if r.Pass {
		t.Error("a method-not-found response must fail methodPresent")
	}
}
