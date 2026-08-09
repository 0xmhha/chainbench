package testspec

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// Derived and chain-specific assertion names.
const (
	assertGasPrice    = "gasPrice"
	assertRPCCall     = "rpcCall"
	assertWSSubscribe = "wsSubscribe"
)

// defaultSubscribeTimeout bounds a WebSocket subscription assertion. Every wait
// in the DSL has a timeout; a subscription that never fires must fail, not hang.
const defaultSubscribeTimeout = 30 * time.Second

// seedDerivedBuiltins registers the transport assertion. rpcCall is registered
// through builtinAssertions instead, because it reads one value like the others
// and therefore has to work as a "read" source too — a name that works in
// "assert" but not in "read" is a trap the spec author only meets at run time.
func seedDerivedBuiltins(r Registry) {
	r.RegisterAssertion(assertWSSubscribe, wsSubscribeAssertion{})
}

// readGasPrice returns eth_gasPrice as a decimal string. Together with baseFee
// it covers the gas-policy cases: on an EIP-1559 chain the suggested price is
// the base fee plus the node's tip, so comparing the two is how a spec checks
// the tip without the harness having to know the chain's tip rule.
func readGasPrice(ctx context.Context, c *rpc.Client, _ map[string]any) (any, error) {
	var s string
	if err := c.Call(ctx, "eth_gasPrice", &s); err != nil {
		return nil, err
	}
	n, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
	if err != nil {
		return nil, fmt.Errorf("testspec: gasPrice: bad value %q: %w", s, err)
	}
	return strconv.FormatUint(n, 10), nil
}

// readRPCCall calls any JSON-RPC method and extracts a value from its result. It
// exists so chain-specific reads — a consensus namespace's
// getWbftExtraInfo.gasTip, istanbul_getValidators, a wemix_ reward query — are
// expressible in a spec without the core learning about any chain. The chain
// vocabulary stays in the spec, where it belongs.
//
// Spec: method (required), params ([]any, optional), select (dot path into the
// result; omitted, the whole result is compared).
func readRPCCall(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	method, _ := spec["method"].(string)
	if method == "" {
		return nil, fmt.Errorf("testspec: rpcCall requires \"method\"")
	}
	var params []any
	if raw, ok := spec["params"].([]any); ok {
		params = raw
	}
	var result any
	if err := c.Call(ctx, method, &result, params...); err != nil {
		return nil, err
	}
	sel, _ := spec["select"].(string)
	if sel == "" {
		return result, nil
	}
	v, ok := dotPath(result, sel)
	if !ok {
		return nil, fmt.Errorf("testspec: rpcCall %s: no %q in the result", method, sel)
	}
	return v, nil
}

// dotPath walks a decoded JSON value by an "a.b.c" path.
func dotPath(v any, path string) (any, bool) {
	cur := v
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		cur, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return cur, true
}

// wsSubscribeAssertion opens a WebSocket subscription and reports how many
// notifications arrived within the window. It proves the WS transport is live,
// which an HTTP-only read cannot: a node can answer eth_blockNumber perfectly
// while its WebSocket endpoint is misconfigured.
//
// Spec: event ("newHeads" by default), params (extra eth_subscribe arguments,
// e.g. a logs filter), count (how many to wait for, default 1), timeout, on.
// A timeout is a failed assertion reporting the count that did arrive, not an
// error — "two heads in five seconds" is a claim that can simply be false.
type wsSubscribeAssertion struct{}

func (wsSubscribeAssertion) Check(ctx context.Context, ac *AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertWSSubscribe, Provenance: ac.Spec}

	wsURL, err := wsTarget(ac)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	event, _ := ac.Spec["event"].(string)
	if event == "" {
		event = "newHeads"
	}
	params := []any{event}
	if extra, ok := ac.Spec["params"].([]any); ok {
		params = append(params, extra...)
	}
	want := 1
	if n, ok := uintArg(ac.Spec["count"]); ok && n > 0 {
		want = int(n)
	}
	res.Expected = ac.Spec["expected"]
	if res.Expected == nil {
		res.Expected = strconv.Itoa(want)
	}

	timeout := durationArg(ac.Spec, "timeout", defaultSubscribeTimeout)
	sctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sub, err := rpc.Subscribe(sctx, wsURL, params...)
	if err != nil {
		res.Actual = err.Error()
		return res, fmt.Errorf("testspec: wsSubscribe %s: %w", event, err)
	}
	defer func() { _ = sub.Close() }()

	got := 0
	for got < want {
		select {
		case msg, ok := <-sub.Notifications():
			if !ok {
				res.Actual = strconv.Itoa(got)
				res.Source = "subscription closed after " + strconv.Itoa(got) + " notification(s)"
				return res, nil
			}
			if len(msg) > 0 && json.Valid(msg) {
				got++
			}
		case <-sctx.Done():
			res.Actual = strconv.Itoa(got)
			res.Source = fmt.Sprintf("only %d of %d %s notification(s) within %s", got, want, event, timeout)
			return res, nil
		}
	}
	res.Actual = strconv.Itoa(got)
	res.Pass = true
	return res, nil
}

// wsTarget derives the WebSocket URL of the assertion's target node from its
// host and WS port. Attached nodes carry no port map, so this names that rather
// than dialing something wrong.
func wsTarget(ac *AssertCtx) (string, error) {
	nodes := ac.On
	if len(nodes) == 0 && ac.Env != nil {
		nodes = ac.Env.Nodes()
	}
	if len(nodes) == 0 {
		return "", fmt.Errorf("testspec: wsSubscribe: no target node")
	}
	n := nodes[0]
	if n.Ports.WS == 0 {
		return "", fmt.Errorf("testspec: wsSubscribe: node%d has no WebSocket port (an attached node's ports are unknown)", n.Index)
	}
	host := n.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("ws://%s:%d", host, n.Ports.WS), nil
}
