package testhelper

import (
	"context"
	"encoding/json"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
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
	sess, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC())
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

func deps() interp.Deps {
	return interp.Deps{RPC: func(u string) *rpc.Client { return rpc.Dial(u) }, Actions: testhelperRegistry()}
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
			r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: tc.spec})
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
	r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: []node.Node{{RPCURL: srv.URL}}, Spec: map[string]any{"assert": assertBalanceAt}})
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
	err := act.Do(context.Background(), &interp.ActionCtx{
		Env:  env,
		Deps: &d,
		Args: map[string]any{"from": "0xabc", "to": "0xdef", "value": "1000", "pollInterval": "5ms"},
	})
	if err != nil {
		t.Fatalf("sendTx: %v", err)
	}
}

func TestSendTxAction_PassesAccessList(t *testing.T) {
	// The empty access list [] is the case a []T-with-omitempty field would drop.
	// Capture the outgoing eth_sendTransaction arg and assert it survived as [].
	var mu sync.Mutex
	var gotParams []json.RawMessage
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int               `json:"id"`
			Method string            `json:"method"`
			Params []json.RawMessage `json:"params"`
		}
		_ = json.Unmarshal(body, &req)
		if req.Method == "eth_sendTransaction" {
			mu.Lock()
			gotParams = req.Params
			mu.Unlock()
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0xhash"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID,
			"result": map[string]any{"status": "0x1", "blockNumber": "0x2"}})
	}))
	t.Cleanup(srv.Close)

	d := deps()
	act, _ := d.Actions.Action(actionSendTx)
	err := act.Do(context.Background(), &interp.ActionCtx{
		Env: envWithNode(t, srv.URL), Deps: &d,
		Args: map[string]any{
			"from": "0xabc", "to": "0xdef", "value": "1", "gasPrice": "1000000000",
			"accessList": []any{}, "pollInterval": "5ms",
		},
	})
	if err != nil {
		t.Fatalf("sendTx: %v", err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gotParams) == 0 {
		t.Fatal("no eth_sendTransaction params captured")
	}
	var arg map[string]any
	if err := json.Unmarshal(gotParams[0], &arg); err != nil {
		t.Fatalf("unmarshal tx arg: %v", err)
	}
	al, ok := arg["accessList"]
	if !ok {
		t.Fatalf("accessList was dropped from the tx arg: %v", arg)
	}
	if _, ok := al.([]any); !ok {
		t.Fatalf("accessList should serialize as an array, got %T", al)
	}
}

func TestSendTxAction_RequiresFrom(t *testing.T) {
	srv := mockRPC(t, map[string]any{})
	d := deps()
	env := envWithNode(t, srv.URL)
	act, _ := d.Actions.Action(actionSendTx)
	if err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{}}); err == nil {
		t.Fatal("expected error when \"from\" missing")
	}
}

func TestSendTxAction_WaitFalseSkipsReceipt(t *testing.T) {
	// No receipt method registered: a wait would 400. wait:false must return early.
	srv := mockRPC(t, map[string]any{"eth_sendTransaction": "0xhash"})
	d := deps()
	env := envWithNode(t, srv.URL)
	act, _ := d.Actions.Action(actionSendTx)
	err := act.Do(context.Background(), &interp.ActionCtx{
		Env:  env,
		Deps: &d,
		Args: map[string]any{"from": "0xabc", "wait": false},
	})
	if err != nil {
		t.Fatalf("sendTx wait:false: %v", err)
	}
}

func TestNewRegistry_BuiltinsGatedByFlag(t *testing.T) {
	if _, ok := interp.NewRegistry().Action(actionSendTx); ok {
		t.Fatal("built-ins must not be seeded when withBuiltins=false")
	}
	if _, ok := testhelperRegistry().Action(actionSendTx); !ok {
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
			r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: tc.spec})
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
		if r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: spec}); err == nil || r.Pass {
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
		r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on(srv.URL), Spec: map[string]any{"assert": assertTxStatus, "hash": "0xh", "expected": "0x1"}})
		if err != nil || !r.Pass {
			t.Fatalf("want pass, got pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
		}
	})
	t.Run("reverted", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_getTransactionReceipt": map[string]any{"status": "0x0"}})
		as, _ := d.Actions.Assertion(assertTxStatus)
		r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on(srv.URL), Spec: map[string]any{"assert": assertTxStatus, "hash": "0xh", "expected": "0x0"}})
		if err != nil || !r.Pass {
			t.Fatalf("want pass for reverted match, got pass=%v err=%v", r.Pass, err)
		}
	})
	t.Run("missing hash", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_getTransactionReceipt": map[string]any{"status": "0x1"}})
		as, _ := d.Actions.Assertion(assertTxStatus)
		if r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on(srv.URL), Spec: map[string]any{"assert": assertTxStatus}}); err == nil || r.Pass {
			t.Fatal("expected error for missing hash")
		}
	})
}

func TestReadReceiptLog(t *testing.T) {
	// A receipt with two logs: a ProposalCreated (topic0=created, topic1=id 1)
	// and an unrelated one, so filtering by topic0/address is exercised.
	created := "0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
	other := "0x82b8cb75fd367be519fd5f57abcb2dbb773381c00082e94059c4713c4dfdfc05"
	id1 := "0x0000000000000000000000000000000000000000000000000000000000000001"
	receipt := map[string]any{"status": "0x1", "logs": []any{
		map[string]any{"address": "0x0000000000000000000000000000000000001003",
			"topics": []any{created, id1, "0xdeadbeef"}, "data": "0xabcd"},
		map[string]any{"address": "0x0000000000000000000000000000000000001003",
			"topics": []any{other, id1}, "data": "0x00"},
	}}
	srv := mockRPC(t, map[string]any{"eth_getTransactionReceipt": receipt})
	c := rpc.Dial(srv.URL)

	// Default: log 0, topic 1 — the proposalId.
	got, err := readReceiptLog(context.Background(), c, map[string]any{"hash": "0xh"})
	if err != nil || got != id1 {
		t.Fatalf("default topic1: got %v err %v", got, err)
	}
	// topic0 filter selects the ProposalCreated log regardless of order.
	got, err = readReceiptLog(context.Background(), c, map[string]any{"hash": "0xh", "topic0": created, "topic": float64(1)})
	if err != nil || got != id1 {
		t.Fatalf("topic0 filter: got %v err %v", got, err)
	}
	// select:"data" returns the log data.
	got, err = readReceiptLog(context.Background(), c, map[string]any{"hash": "0xh", "select": "data"})
	if err != nil || got != "0xabcd" {
		t.Fatalf("select data: got %v err %v", got, err)
	}
	// A topic index past the end is an error, not a panic.
	if _, err := readReceiptLog(context.Background(), c, map[string]any{"hash": "0xh", "topic0": other, "topic": float64(5)}); err == nil {
		t.Fatal("out-of-range topic must fail")
	}
	// Missing hash is an error.
	if _, err := readReceiptLog(context.Background(), c, map[string]any{}); err == nil {
		t.Fatal("missing hash must fail")
	}
}

func TestWaitBlockAction(t *testing.T) {
	d := deps()

	t.Run("target reached", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x10"}) // 16
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionWaitBlock)
		if err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{"target": float64(5), "pollInterval": "5ms"}}); err != nil {
			t.Fatalf("waitBlock: %v", err)
		}
	})
	t.Run("timeout when unreached", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x1"}) // 1
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionWaitBlock)
		err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{"target": float64(100), "timeout": "40ms", "pollInterval": "5ms"}})
		if err == nil {
			t.Fatal("expected timeout error for unreachable target")
		}
	})
	t.Run("requires target", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x10"})
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionWaitBlock)
		if err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{}}); err == nil {
			t.Fatal("expected error for missing target")
		}
	})
}

func TestWaitForAction(t *testing.T) {
	d := deps()

	t.Run("condition met", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x10"}) // 16
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionWaitFor)
		ac := &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{
			"source": assertBlockNumber, "compare": "GreaterOrEqual", "expected": float64(5), "pollInterval": "5ms"}}
		if err := act.Do(context.Background(), ac); err != nil {
			t.Fatalf("waitFor: %v", err)
		}
		if ac.Value != "16" {
			t.Fatalf("bound value = %v, want the satisfying read \"16\"", ac.Value)
		}
	})
	t.Run("timeout when unmet", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x1"}) // 1
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionWaitFor)
		err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{
			"source": assertBlockNumber, "compare": "GreaterOrEqual", "expected": float64(100), "timeout": "40ms", "pollInterval": "5ms"}})
		if err == nil {
			t.Fatal("expected timeout error for a condition that never holds")
		}
	})
	t.Run("requires source", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x10"})
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionWaitFor)
		if err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{}}); err == nil {
			t.Fatal("expected error for missing source")
		}
	})
	t.Run("unknown comparator", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x10"})
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionWaitFor)
		err := act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: map[string]any{
			"source": assertBlockNumber, "compare": "Nonsense", "expected": float64(1)}})
		if err == nil {
			t.Fatal("expected error for an unknown comparator")
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
	err := act.Do(context.Background(), &interp.ActionCtx{
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
		return act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: args})
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

// mockRPCReject serves a JSON-RPC error for eth_sendTransaction, simulating a
// node that refuses the transaction at submit time (no hash returned).
func mockRPCReject(t *testing.T, message string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID int `json:"id"`
		}
		_ = json.Unmarshal(body, &req)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"jsonrpc": "2.0", "id": req.ID,
			"error": map[string]any{"code": -32000, "message": message},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestSendTxAction_ExpectReject(t *testing.T) {
	d := deps()
	run := func(srv *httptest.Server, args map[string]any) error {
		env := envWithNode(t, srv.URL)
		act, _ := d.Actions.Action(actionSendTx)
		return act.Do(context.Background(), &interp.ActionCtx{Env: env, Deps: &d, Args: args})
	}

	// expect:"reject" + submit error => step passes.
	rej := mockRPCReject(t, "insufficient funds for gas * price + value")
	if err := run(rej, map[string]any{"from": "0xa", "expect": "reject"}); err != nil {
		t.Fatalf("expect:reject on a rejected submit should pass: %v", err)
	}

	// expectReject:true alias, same rejection => passes.
	if err := run(rej, map[string]any{"from": "0xa", "expectReject": true}); err != nil {
		t.Fatalf("expectReject alias should pass on a rejected submit: %v", err)
	}

	// expect:"reject" but the node accepts the tx => step fails.
	ok := mockRPC(t, map[string]any{"eth_sendTransaction": "0xhash"})
	if err := run(ok, map[string]any{"from": "0xa", "expect": "reject", "wait": false}); err == nil {
		t.Fatal("expect:reject must fail the step when the submit is accepted")
	}

	// reason substring matches the rejection message => passes.
	if err := run(rej, map[string]any{"from": "0xa", "expect": "reject", "reason": "insufficient funds"}); err != nil {
		t.Fatalf("matching reason should pass: %v", err)
	}

	// reason substring does not match => step fails (rejected, wrong reason).
	if err := run(rej, map[string]any{"from": "0xa", "expect": "reject", "reason": "nonce too low"}); err == nil {
		t.Fatal("a non-matching reason must fail the step")
	}
}

func TestNewAccountAction_BindsAddressAndKey(t *testing.T) {
	d := deps()
	act, ok := d.Actions.Action(actionNewAccount)
	if !ok {
		t.Fatal("newAccount not registered")
	}
	ac := &interp.ActionCtx{Args: map[string]any{"save": "acct", "saveKey": "acctKey"}}
	if err := act.Do(context.Background(), ac); err != nil {
		t.Fatalf("newAccount: %v", err)
	}
	addr, _ := ac.Value.(string)
	if !strings.HasPrefix(addr, "0x") || len(addr) != 42 {
		t.Fatalf("address = %q, want 0x + 40 hex", addr)
	}
	key, _ := ac.Extra["acctKey"].(string)
	if len(key) != 64 {
		t.Fatalf("private key hex = %q (len %d), want 64", key, len(key))
	}
}

func TestNewAccountAction_RequiresSaveKey(t *testing.T) {
	d := deps()
	act, _ := d.Actions.Action(actionNewAccount)
	if err := act.Do(context.Background(), &interp.ActionCtx{Args: map[string]any{"save": "acct"}}); err == nil {
		t.Fatal("newAccount without saveKey must error")
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
	r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: spec})
	if err != nil || !r.Pass {
		t.Fatalf("onEach all-pass: pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
	}

	// One node reports 0 peers -> the assertion fails and names that node.
	srvBad := mockRPC(t, map[string]any{"net_peerCount": "0x0"})
	onBad := []node.Node{{Index: 1, RPCURL: srv1.URL}, {Index: 2, RPCURL: srvBad.URL}}
	r, err = as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: onBad, Spec: spec})
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
		r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}},
			Spec: map[string]any{"assert": assertBlockAdvance, "pollInterval": "5ms", "timeout": "2s"}})
		if err != nil || !r.Pass {
			t.Fatalf("expected advance pass: pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
		}
	})

	t.Run("stalled fails", func(t *testing.T) {
		srv := mockRPC(t, map[string]any{"eth_blockNumber": "0x5"}) // never changes
		r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}},
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
		r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: map[string]any{"assert": assertSameBlockHash, "block": "0x0"}})
		if err != nil || !r.Pass {
			t.Fatalf("agreeing nodes should pass: pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
		}
	})

	t.Run("fork", func(t *testing.T) {
		a, b := mk("0xaaaa"), mk("0xbbbb")
		on := []node.Node{{Index: 1, RPCURL: a.URL}, {Index: 2, RPCURL: b.URL}}
		r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: map[string]any{"assert": assertSameBlockHash, "block": "0x0"}})
		if err != nil {
			t.Fatalf("fork check should not error: %v", err)
		}
		if r.Pass {
			t.Fatalf("divergent hashes must fail: %+v", r)
		}
	})
}

func TestBaseFeeAssertion(t *testing.T) {
	// baseFeePerGas = 20000000000000 (0x12309ce54000)
	srv := mockRPC(t, map[string]any{"eth_getBlockByNumber": map[string]any{"number": "0x1", "hash": "0xh", "baseFeePerGas": "0x12309ce54000"}})
	d := deps()
	as, _ := d.Actions.Assertion(assertBaseFee)
	on := []node.Node{{Index: 1, RPCURL: srv.URL}}

	// >= minimum passes
	r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: map[string]any{"assert": assertBaseFee, "expected": "20000000000000"}})
	if err != nil || !r.Pass {
		t.Fatalf("baseFee >= min: pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
	}
	// <= maximum passes
	r, err = as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: map[string]any{"assert": assertBaseFee, "expected": "20000000000000000", "compare": "LessOrEqual"}})
	if err != nil || !r.Pass {
		t.Fatalf("baseFee <= max: pass=%v err=%v", r.Pass, err)
	}
	// below a higher minimum fails
	r, err = as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: map[string]any{"assert": assertBaseFee, "expected": "99000000000000"}})
	if err != nil || r.Pass {
		t.Fatalf("baseFee below min should fail: pass=%v", r.Pass)
	}
}

func TestEstimateGasAssertion(t *testing.T) {
	srv := mockRPC(t, map[string]any{"eth_estimateGas": "0x8000"}) // 32768
	d := deps()
	as, _ := d.Actions.Assertion(assertEstimateGas)
	on := []node.Node{{Index: 1, RPCURL: srv.URL}}

	// exceeds 21000
	spec := map[string]any{"assert": assertEstimateGas, "to": "0xc0", "data": "0xa9059cbb", "expected": float64(21000), "compare": "Greater"}
	r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: spec})
	if err != nil || !r.Pass {
		t.Fatalf("estimateGas > 21000: pass=%v err=%v actual=%v", r.Pass, err, r.Actual)
	}
	// missing data errors
	if r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: on, Spec: map[string]any{"assert": assertEstimateGas, "to": "0xc0"}}); err == nil || r.Pass {
		t.Fatal("estimateGas requires data")
	}
}
