package testspec

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// mockRPC serves canned JSON-RPC results keyed by method.
func mockRPC(t *testing.T, results map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		res, ok := results[req.Method]
		if !ok {
			http.Error(w, "unknown method "+req.Method, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": res})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// envWithNode builds a real session environment whose primary node points at url.
func envWithNode(t *testing.T, url string) session.Environment {
	t.Helper()
	sess, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC(), nil)
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	env, err := sess.NewEnvironment("aaaaaaaaaaaa0000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	env.PopulateNodeTable(node.NodeSet{Nodes: []node.Node{
		{Index: 1, Role: node.RoleValidator, Host: "127.0.0.1", RPCURL: url},
	}})
	return env
}

func deps() Deps {
	return Deps{RPC: func(u string) *rpc.Client { return rpc.Dial(u) }, Actions: NewRegistry(true)}
}

func TestBuiltinAssertions(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"eth_chainId":     "0x539",             // 1337
		"eth_blockNumber": "0x10",              // 16
		"net_peerCount":   "0x3",               // 3
		"eth_getBalance":  "0xde0b6b3a7640000", // 1e18
		"eth_getCode":     "0x6060",
	})
	d := deps()
	reg := d.Actions
	on := []node.Node{{Index: 1, RPCURL: srv.URL}}

	cases := []struct {
		name string
		spec map[string]any
		pass bool
	}{
		{"chainId match", map[string]any{"assert": assertChainID, "expected": float64(1337)}, true},
		{"chainId mismatch", map[string]any{"assert": assertChainID, "expected": float64(1)}, false},
		{"blockNumber ge default", map[string]any{"assert": assertBlockNumber, "expected": float64(10)}, true},
		{"blockNumber ge fail", map[string]any{"assert": assertBlockNumber, "expected": float64(99)}, false},
		{"peerCount ge", map[string]any{"assert": assertPeerCount, "expected": float64(3)}, true},
		{"balanceAt equal", map[string]any{"assert": assertBalanceAt, "address": "0xabc", "expected": "1000000000000000000"}, true},
		{"balanceAt compare gt", map[string]any{"assert": assertBalanceAt, "address": "0xabc", "expected": float64(0), "compare": "Greater"}, true},
		{"codeAt equal", map[string]any{"assert": assertCodeAt, "address": "0xabc", "expected": "0x6060"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			as, ok := reg.Assertion(tc.spec["assert"].(string))
			if !ok {
				t.Fatalf("assertion %q not registered", tc.spec["assert"])
			}
			r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: on, Spec: tc.spec})
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if r.Pass != tc.pass {
				t.Fatalf("pass = %v, want %v (actual=%v)", r.Pass, tc.pass, r.Actual)
			}
		})
	}
}

func TestBuiltinAssertion_MissingAddress(t *testing.T) {
	srv := mockRPC(t, map[string]any{"eth_getBalance": "0x1"})
	d := deps()
	as, _ := d.Actions.Assertion(assertBalanceAt)
	r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: []node.Node{{RPCURL: srv.URL}}, Spec: map[string]any{"assert": assertBalanceAt}})
	if err == nil {
		t.Fatal("expected error for missing address")
	}
	if r.Pass {
		t.Fatal("result must not pass on error")
	}
}

func TestSendTxAction_SendsAndWaits(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"eth_sendTransaction":       "0xhash",
		"eth_getTransactionReceipt": map[string]any{"status": "0x1", "blockNumber": "0x2"},
	})
	d := deps()
	env := envWithNode(t, srv.URL)
	act, ok := d.Actions.Action(actionSendTx)
	if !ok {
		t.Fatal("sendTx not registered")
	}
	err := act.Do(context.Background(), &ActionCtx{
		Env:  env,
		Deps: &d,
		Args: map[string]any{"from": "0xabc", "to": "0xdef", "value": "1000", "pollInterval": "5ms"},
	})
	if err != nil {
		t.Fatalf("sendTx: %v", err)
	}
}

func TestSendTxAction_RequiresFrom(t *testing.T) {
	srv := mockRPC(t, map[string]any{})
	d := deps()
	env := envWithNode(t, srv.URL)
	act, _ := d.Actions.Action(actionSendTx)
	if err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{}}); err == nil {
		t.Fatal("expected error when \"from\" missing")
	}
}

func TestSendTxAction_WaitFalseSkipsReceipt(t *testing.T) {
	// No receipt method registered: a wait would 400. wait:false must return early.
	srv := mockRPC(t, map[string]any{"eth_sendTransaction": "0xhash"})
	d := deps()
	env := envWithNode(t, srv.URL)
	act, _ := d.Actions.Action(actionSendTx)
	err := act.Do(context.Background(), &ActionCtx{
		Env:  env,
		Deps: &d,
		Args: map[string]any{"from": "0xabc", "wait": false},
	})
	if err != nil {
		t.Fatalf("sendTx wait:false: %v", err)
	}
}

func TestNewRegistry_BuiltinsGatedByFlag(t *testing.T) {
	if _, ok := NewRegistry(false).Action(actionSendTx); ok {
		t.Fatal("built-ins must not be seeded when withBuiltins=false")
	}
	if _, ok := NewRegistry(true).Action(actionSendTx); !ok {
		t.Fatal("built-ins must be seeded when withBuiltins=true")
	}
}

func TestBuiltinAssertions_NonceAndCall(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"eth_getTransactionCount": "0x5",  // nonce 5
		"eth_call":                "0x2a", // call result
	})
	d := deps()
	on := []node.Node{{Index: 1, RPCURL: srv.URL}}

	cases := []struct {
		name string
		spec map[string]any
		pass bool
	}{
		{"nonceAt equal", map[string]any{"assert": assertNonceAt, "address": "0xabc", "expected": float64(5)}, true},
		{"nonceAt mismatch", map[string]any{"assert": assertNonceAt, "address": "0xabc", "expected": float64(4)}, false},
		{"nonceAt compare ge", map[string]any{"assert": assertNonceAt, "address": "0xabc", "expected": float64(1), "compare": "GreaterOrEqual"}, true},
		{"call equal", map[string]any{"assert": assertCall, "to": "0xc0", "data": "0xdead", "expected": "0x2a"}, true},
		{"call mismatch", map[string]any{"assert": assertCall, "to": "0xc0", "data": "0xdead", "expected": "0x2b"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			as, ok := d.Actions.Assertion(tc.spec["assert"].(string))
			if !ok {
				t.Fatalf("assertion %q not registered", tc.spec["assert"])
			}
			r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: on, Spec: tc.spec})
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if r.Pass != tc.pass {
				t.Fatalf("pass = %v, want %v (actual=%v)", r.Pass, tc.pass, r.Actual)
			}
		})
	}
}

func TestBuiltinAssertions_NonceCallMissingArgs(t *testing.T) {
	srv := mockRPC(t, map[string]any{"eth_getTransactionCount": "0x1", "eth_call": "0x1"})
	d := deps()
	on := []node.Node{{RPCURL: srv.URL}}
	for _, spec := range []map[string]any{
		{"assert": assertNonceAt},                // missing address
		{"assert": assertCall, "to": "0xc0"},     // missing data
		{"assert": assertCall, "data": "0xdead"}, // missing to
	} {
		as, _ := d.Actions.Assertion(spec["assert"].(string))
		if r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: on, Spec: spec}); err == nil || r.Pass {
			t.Fatalf("expected error for %v", spec)
		}
	}
}
