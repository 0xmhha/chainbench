package testspec

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
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

func TestBuiltinAssertion_TxStatus(t *testing.T) {
	d := deps()
	on := func(url string) []node.Node { return []node.Node{{RPCURL: url}} }

	t.Run("success", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_getTransactionReceipt": map[string]any{"status": "0x1"}})
		as, _ := d.Actions.Assertion(assertTxStatus)
		r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: on(srv.URL), Spec: map[string]any{"assert": assertTxStatus, "hash": "0xh", "expected": "0x1"}})
		if err != nil || !r.Pass {
			t.Fatalf("want pass, got pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
		}
	})
	t.Run("reverted", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_getTransactionReceipt": map[string]any{"status": "0x0"}})
		as, _ := d.Actions.Assertion(assertTxStatus)
		r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: on(srv.URL), Spec: map[string]any{"assert": assertTxStatus, "hash": "0xh", "expected": "0x0"}})
		if err != nil || !r.Pass {
			t.Fatalf("want pass for reverted match, got pass=%v err=%v", r.Pass, err)
		}
	})
	t.Run("missing hash", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_getTransactionReceipt": map[string]any{"status": "0x1"}})
		as, _ := d.Actions.Assertion(assertTxStatus)
		if r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: on(srv.URL), Spec: map[string]any{"assert": assertTxStatus}}); err == nil || r.Pass {
			t.Fatal("expected error for missing hash")
		}
	})
}

func TestWaitBlockAction(t *testing.T) {
	d := deps()

	t.Run("target reached", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x10"}) // 16
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionWaitBlock)
		if err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{"target": float64(5), "pollInterval": "5ms"}}); err != nil {
			t.Fatalf("waitBlock: %v", err)
		}
	})
	t.Run("timeout when unreached", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x1"}) // 1
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionWaitBlock)
		err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{"target": float64(100), "timeout": "40ms", "pollInterval": "5ms"}})
		if err == nil {
			t.Fatal("expected timeout error for unreachable target")
		}
	})
	t.Run("requires target", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x10"})
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionWaitBlock)
		if err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{}}); err == nil {
			t.Fatal("expected error for missing target")
		}
	})
}

func TestSendTxAction_RevertFailsStep(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"eth_sendTransaction":       "0xhash",
		"eth_getTransactionReceipt": map[string]any{"status": "0x0"}, // reverted
	})
	d := deps()
	env := envWithNode(t, srv.URL)
	act, _ := d.Actions.Action(actionSendTx)
	err := act.Do(context.Background(), &ActionCtx{
		Env:  env,
		Deps: &d,
		Args: map[string]any{"from": "0xabc", "to": "0xdef", "pollInterval": "5ms"},
	})
	if err == nil {
		t.Fatal("a reverted tx must fail the step by default")
	}
}

func TestSendTxAction_ExpectRevert(t *testing.T) {
	mk := func(status string) *httptest.Server {
		return mockRPC(t, map[string]any{
			"eth_sendTransaction":       "0xhash",
			"eth_getTransactionReceipt": map[string]any{"status": status},
		})
	}
	d := deps()
	run := func(srv *httptest.Server, args map[string]any) error {
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionSendTx)
		return act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: args})
	}

	// expectRevert + revert => step passes.
	if err := run(mk("0x0"), map[string]any{"from": "0xa", "expectRevert": true, "pollInterval": "5ms"}); err != nil {
		t.Fatalf("expectRevert on revert should pass: %v", err)
	}
	// expectRevert + success => step fails.
	if err := run(mk("0x1"), map[string]any{"from": "0xa", "expectRevert": true, "pollInterval": "5ms"}); err == nil {
		t.Fatal("expectRevert on success must fail the step")
	}
	// expect:"revert" alias.
	if err := run(mk("0x0"), map[string]any{"from": "0xa", "expect": "revert", "pollInterval": "5ms"}); err != nil {
		t.Fatalf("expect:revert alias should pass on revert: %v", err)
	}
}

func TestRPCAssertion_OnEachAllNodes(t *testing.T) {
	// Two nodes, each with peerCount 2; onEach must check both.
	srv1 := mockRPC(t, map[string]any{"net_peerCount": "0x2"})
	srv2 := mockRPC(t, map[string]any{"net_peerCount": "0x2"})
	d := deps()
	as, _ := d.Actions.Assertion(assertPeerCount)

	on := []node.Node{{Index: 1, RPCURL: srv1.URL}, {Index: 2, RPCURL: srv2.URL}}
	spec := map[string]any{"assert": assertPeerCount, "expected": float64(1), "compare": "GreaterOrEqual"}
	r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: on, Spec: spec})
	if err != nil || !r.Pass {
		t.Fatalf("onEach all-pass: pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
	}

	// One node reports 0 peers -> the assertion fails and names that node.
	srvBad := mockRPC(t, map[string]any{"net_peerCount": "0x0"})
	onBad := []node.Node{{Index: 1, RPCURL: srv1.URL}, {Index: 2, RPCURL: srvBad.URL}}
	r, err = as.Check(context.Background(), &AssertCtx{Deps: &d, On: onBad, Spec: spec})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Pass {
		t.Fatalf("onEach must fail when one node fails: %+v", r)
	}
	if !strings.Contains(r.Source, "node2") {
		t.Fatalf("failure should name the failing node, got Source=%q", r.Source)
	}
}

func TestBlockAdvanceAssertion(t *testing.T) {
	d := deps()
	as, _ := d.Actions.Assertion(assertBlockAdvance)

	t.Run("advances", func(t *testing.T) {
		var mu sync.Mutex
		var n int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				ID int `json:"id"`
			}
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &req)
			mu.Lock()
			n++ // 1,2,3,... so the head keeps advancing
			cur := n
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x" + strconv.FormatInt(int64(cur), 16)})
		}))
		defer srv.Close()
		r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}},
			Spec: map[string]any{"assert": assertBlockAdvance, "pollInterval": "5ms", "timeout": "2s"}})
		if err != nil || !r.Pass {
			t.Fatalf("expected advance pass: pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
		}
	})

	t.Run("stalled fails", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x5"}) // never changes
		r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}},
			Spec: map[string]any{"assert": assertBlockAdvance, "pollInterval": "5ms", "timeout": "40ms"}})
		if err != nil {
			t.Fatalf("stalled should not error: %v", err)
		}
		if r.Pass {
			t.Fatal("a stalled head must fail blockAdvance")
		}
	})
}

func TestSameBlockHashAssertion(t *testing.T) {
	d := deps()
	as, _ := d.Actions.Assertion(assertSameBlockHash)
	mk := func(hash string) *httptest.Server {
		return mockRPC(t, map[string]any{"eth_getBlockByNumber": map[string]any{"number": "0x0", "hash": hash}})
	}

	t.Run("agree", func(t *testing.T) {
		a, b := mk("0xsame"), mk("0xsame")
		on := []node.Node{{Index: 1, RPCURL: a.URL}, {Index: 2, RPCURL: b.URL}}
		r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: on, Spec: map[string]any{"assert": assertSameBlockHash, "block": "0x0"}})
		if err != nil || !r.Pass {
			t.Fatalf("agreeing nodes should pass: pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
		}
	})

	t.Run("fork", func(t *testing.T) {
		a, b := mk("0xaaaa"), mk("0xbbbb")
		on := []node.Node{{Index: 1, RPCURL: a.URL}, {Index: 2, RPCURL: b.URL}}
		r, err := as.Check(context.Background(), &AssertCtx{Deps: &d, On: on, Spec: map[string]any{"assert": assertSameBlockHash, "block": "0x0"}})
		if err != nil {
			t.Fatalf("fork check should not error: %v", err)
		}
		if r.Pass {
			t.Fatalf("divergent hashes must fail: %+v", r)
		}
	})
}
