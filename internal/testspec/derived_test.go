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
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/gorilla/websocket"
)

func TestGasPriceAssertion(t *testing.T) {
	srv := mockRPC(t, map[string]any{"eth_gasPrice": "0x3b9aca00"}) // 1 gwei
	d := deps()
	as, ok := d.Actions.Assertion(assertGasPrice)
	if !ok {
		t.Fatal("gasPrice not registered")
	}
	res, err := as.Check(context.Background(), &AssertCtx{
		Env: envWithNode(t, srv.URL), Deps: &d,
		Spec: map[string]any{"assert": assertGasPrice, "expected": "1000000000"},
	})
	if err != nil {
		t.Fatalf("gasPrice: %v", err)
	}
	if !res.Pass {
		t.Fatalf("actual %#v", res.Actual)
	}
}

func TestRPCCallAssertion_ReadsAChainSpecificMethod(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"istanbul_getWbftExtraInfo": map[string]any{
			"gasTip":    "0x64",
			"epochInfo": map[string]any{"stabilizing": true},
		},
	})
	d := deps()
	as, ok := d.Actions.Assertion(assertRPCCall)
	if !ok {
		t.Fatal("rpcCall not registered")
	}

	// A top-level field.
	res, err := as.Check(context.Background(), &AssertCtx{
		Env: envWithNode(t, srv.URL), Deps: &d,
		Spec: map[string]any{
			"assert": assertRPCCall, "method": "istanbul_getWbftExtraInfo",
			"params": []any{"latest"}, "select": "gasTip", "expected": "0x64",
		},
	})
	if err != nil {
		t.Fatalf("rpcCall: %v", err)
	}
	if !res.Pass {
		t.Fatalf("actual %#v", res.Actual)
	}

	// A nested field, via a dot path.
	res, err = as.Check(context.Background(), &AssertCtx{
		Env: envWithNode(t, srv.URL), Deps: &d,
		Spec: map[string]any{
			"assert": assertRPCCall, "method": "istanbul_getWbftExtraInfo",
			"select": "epochInfo.stabilizing", "expected": true,
		},
	})
	if err != nil {
		t.Fatalf("rpcCall nested: %v", err)
	}
	if !res.Pass {
		t.Fatalf("nested actual %#v", res.Actual)
	}
}

func TestRPCCallAssertion_MissingSelectPathIsAnError(t *testing.T) {
	srv := mockRPC(t, map[string]any{"istanbul_status": map[string]any{"a": 1}})
	d := deps()
	as, _ := d.Actions.Assertion(assertRPCCall)
	_, err := as.Check(context.Background(), &AssertCtx{
		Env: envWithNode(t, srv.URL), Deps: &d,
		Spec: map[string]any{"assert": assertRPCCall, "method": "istanbul_status", "select": "b.c", "expected": 1},
	})
	if err == nil {
		t.Fatal("expected an error for a select path that is not in the result")
	}
}

func TestRPCCallAssertion_RequiresAMethod(t *testing.T) {
	d := deps()
	as, _ := d.Actions.Assertion(assertRPCCall)
	if _, err := as.Check(context.Background(), &AssertCtx{
		Env: envWithNode(t, "http://unused"), Deps: &d,
		Spec: map[string]any{"assert": assertRPCCall, "expected": 1},
	}); err == nil {
		t.Fatal("expected an error without a method")
	}
}

func TestRPCCallAssertion_IndexesArraysAndReportsLength(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"admin_peers": []any{
			map[string]any{"id": "enode://abc"},
			map[string]any{"id": "enode://def"},
		},
	})
	d := deps()
	as, _ := d.Actions.Assertion(assertRPCCall)

	// "#" yields the array length (decimal), so an "at least one" check works.
	res, err := as.Check(context.Background(), &AssertCtx{
		Env: envWithNode(t, srv.URL), Deps: &d,
		Spec: map[string]any{
			"assert": assertRPCCall, "method": "admin_peers",
			"select": "#", "compare": "GreaterOrEqual", "expected": "1",
		},
	})
	if err != nil {
		t.Fatalf("length select: %v", err)
	}
	if !res.Pass {
		t.Fatalf("length actual %#v", res.Actual)
	}

	// A numeric segment indexes into the array, then walks into the element.
	res, err = as.Check(context.Background(), &AssertCtx{
		Env: envWithNode(t, srv.URL), Deps: &d,
		Spec: map[string]any{
			"assert": assertRPCCall, "method": "admin_peers",
			"select": "0.id", "compare": "NotEqual", "expected": "",
		},
	})
	if err != nil {
		t.Fatalf("index select: %v", err)
	}
	if !res.Pass {
		t.Fatalf("index actual %#v", res.Actual)
	}
}

func TestRPCCallAssertion_OutOfRangeIndexIsAnError(t *testing.T) {
	srv := mockRPC(t, map[string]any{"admin_peers": []any{}})
	d := deps()
	as, _ := d.Actions.Assertion(assertRPCCall)
	if _, err := as.Check(context.Background(), &AssertCtx{
		Env: envWithNode(t, srv.URL), Deps: &d,
		Spec: map[string]any{"assert": assertRPCCall, "method": "admin_peers", "select": "0.id", "expected": "x"},
	}); err == nil {
		t.Fatal("expected an error indexing an empty array")
	}
}

// wsHeadServer serves eth_subscribe over a WebSocket and pushes n newHeads
// notifications.
func wsHeadServer(t *testing.T, n int) *httptest.Server {
	t.Helper()
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		var req struct {
			ID int `json:"id"`
		}
		if err := conn.ReadJSON(&req); err != nil {
			return
		}
		_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0xsub"})
		for i := 0; i < n; i++ {
			_ = conn.WriteJSON(map[string]any{
				"jsonrpc": "2.0", "method": "eth_subscription",
				"params": map[string]any{"subscription": "0xsub", "result": map[string]any{"number": "0x1"}},
			})
		}
		// Hold the connection open until the client hangs up.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// envWithWS builds an environment whose node's WS endpoint is the given server.
func envWithWS(t *testing.T, host string, wsPort int) session.Environment {
	t.Helper()
	sess, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC(), nil)
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	env, err := sess.NewEnvironment("dddddddddddd0000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	env.PopulateNodeTable(node.NodeSet{Nodes: []node.Node{
		{Index: 1, Role: node.RoleValidator, Host: host, Ports: node.Endpoints{WS: wsPort}},
	}})
	return env
}

func TestWSSubscribeAssertion_CountsNotifications(t *testing.T) {
	srv := wsHeadServer(t, 3)
	host, port := hostPort(t, srv.URL)
	d := deps()
	as, ok := d.Actions.Assertion(assertWSSubscribe)
	if !ok {
		t.Fatal("wsSubscribe not registered")
	}
	res, err := as.Check(context.Background(), &AssertCtx{
		Env: envWithWS(t, host, port), Deps: &d,
		Spec: map[string]any{
			"assert": assertWSSubscribe, "event": "newHeads",
			"count": 2, "timeout": "5s", "expected": "2",
		},
	})
	if err != nil {
		t.Fatalf("wsSubscribe: %v", err)
	}
	if !res.Pass {
		t.Fatalf("actual %#v (%s)", res.Actual, res.Source)
	}
}

func TestWSSubscribeAssertion_TimeoutReportsWhatArrived(t *testing.T) {
	srv := wsHeadServer(t, 0)
	host, port := hostPort(t, srv.URL)
	d := deps()
	as, _ := d.Actions.Assertion(assertWSSubscribe)
	res, err := as.Check(context.Background(), &AssertCtx{
		Env: envWithWS(t, host, port), Deps: &d,
		Spec: map[string]any{
			"assert": assertWSSubscribe, "event": "newHeads",
			"count": 1, "timeout": "300ms", "expected": "1",
		},
	})
	if err != nil {
		t.Fatalf("a timeout should be a failed assertion, not an error: %v", err)
	}
	if res.Pass {
		t.Fatal("expected a failure when no notification arrived")
	}
	if res.Actual != "0" {
		t.Fatalf("actual = %#v, want the count that did arrive", res.Actual)
	}
}

// hostPort splits an httptest URL into host and port.
func hostPort(t *testing.T, rawURL string) (string, int) {
	t.Helper()
	var host string
	var port int
	if _, err := fmtSscan(rawURL, &host, &port); err != nil {
		t.Fatalf("parse %s: %v", rawURL, err)
	}
	return host, port
}

// fmtSscan parses "http://host:port" into its parts.
func fmtSscan(rawURL string, host *string, port *int) (int, error) {
	u := rawURL
	u = trimPrefix(u, "http://")
	for i := len(u) - 1; i >= 0; i-- {
		if u[i] == ':' {
			*host = u[:i]
			var err error
			*port, err = atoi(u[i+1:])
			return 2, err
		}
	}
	return 0, io.ErrUnexpectedEOF
}

func trimPrefix(s, p string) string {
	if len(s) >= len(p) && s[:len(p)] == p {
		return s[len(p):]
	}
	return s
}

func atoi(s string) (int, error) {
	var n int
	if err := json.Unmarshal([]byte(s), &n); err != nil {
		return 0, err
	}
	return n, nil
}

func TestReadDerive(t *testing.T) {
	cases := []struct {
		spec map[string]any
		want string
	}{
		{map[string]any{"op": "sum", "of": []any{"0x10", "0x6"}}, "22"},
		{map[string]any{"op": "sum", "of": []any{"100", float64(11)}}, "111"},
		{map[string]any{"op": "diff", "of": []any{"0x64", "40", "0x4"}}, "56"},
	}
	for _, tc := range cases {
		got, err := readDerive(context.Background(), nil, tc.spec)
		if err != nil {
			t.Fatalf("%v: %v", tc.spec, err)
		}
		if got != tc.want {
			t.Errorf("%v = %v, want %v", tc.spec, got, tc.want)
		}
	}

	for name, bad := range map[string]map[string]any{
		"no of":      {"op": "sum"},
		"unknown op": {"op": "mul", "of": []any{"1", "2"}},
		"bad value":  {"op": "sum", "of": []any{"zzz"}},
	} {
		if _, err := readDerive(context.Background(), nil, bad); err == nil {
			t.Errorf("%s must fail", name)
		}
	}
}

func TestReadDerive_AbiCall(t *testing.T) {
	cases := []struct {
		spec map[string]any
		want string
	}{
		// approveProposal(1): selector + a single left-padded uint256 word.
		{map[string]any{"op": "abiCall", "selector": "0x98951b56", "of": []any{"1"}},
			"0x98951b560000000000000000000000000000000000000000000000000000000000000001"},
		// A 32-byte 0x-hex topic (as receiptLog yields) parses to the same word.
		{map[string]any{"op": "abiCall", "selector": "0x0d61b519",
			"of": []any{"0x0000000000000000000000000000000000000000000000000000000000000002"}},
			"0x0d61b5190000000000000000000000000000000000000000000000000000000000000002"},
		// An address argument is right-aligned in its word, same as a uint.
		{map[string]any{"op": "abiCall", "selector": "0xa9059cbb",
			"of": []any{"0x00000000000000000000000000000000c0ffee05", "1"}},
			"0xa9059cbb00000000000000000000000000000000000000000000000000000000c0ffee050000000000000000000000000000000000000000000000000000000000000001"},
		// No args is a bare selector.
		{map[string]any{"op": "abiCall", "selector": "0x12345678"}, "0x12345678"},
	}
	for _, tc := range cases {
		got, err := readDerive(context.Background(), nil, tc.spec)
		if err != nil {
			t.Fatalf("%v: %v", tc.spec, err)
		}
		if got != tc.want {
			t.Errorf("%v =\n %v\nwant\n %v", tc.spec, got, tc.want)
		}
	}

	for name, bad := range map[string]map[string]any{
		"no selector":    {"op": "abiCall", "of": []any{"1"}},
		"short selector": {"op": "abiCall", "selector": "0x1234"},
		"bad arg":        {"op": "abiCall", "selector": "0x98951b56", "of": []any{"zzz"}},
	} {
		if _, err := readDerive(context.Background(), nil, bad); err == nil {
			t.Errorf("%s must fail", name)
		}
	}
}
