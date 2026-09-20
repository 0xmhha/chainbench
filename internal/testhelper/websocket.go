package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/0xmhha/chainbench/internal/dsl/interp"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// The WebSocket vocabulary: opening a subscription, collecting what it pushes,
// and asserting on the result.
//
// A subscription dials the endpoint the composition recorded rather than one
// built from the node's own address — under docker those differ, and building
// it here is what made this the one dial that skipped the translation.

// be open before the transaction that emits the log, because eth_subscribe does
// not backfill. Args: event ("logs" by default), address / topics (a logs
// filter), params (extra eth_subscribe args, verbatim), on, save (required).
type wsOpenAction struct{}

func (wsOpenAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	if s, _ := ac.Args["save"].(string); s == "" {
		return fmt.Errorf("dsl: wsOpen requires \"save\" to bind the subscription handle")
	}
	wsURL, err := wsTargetURL(nodesFor(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	args, aerr := resolveAddressArgs(ac.Deps, ac.Args)
	if aerr != nil {
		return aerr
	}
	event, _ := ac.Args["event"].(string)
	if event == "" {
		event = "logs"
	}
	params := []any{event}
	filter, ferr := logsFilter(ac.Deps, ac.Args)
	if ferr != nil {
		return ferr
	}
	if filter != nil {
		params = append(params, filter)
	}
	if extra, ok := args["params"].([]any); ok {
		params = append(params, extra...)
	}
	sub, err := rpc.Subscribe(ctx, wsURL, params...)
	if err != nil {
		return fmt.Errorf("dsl: wsOpen %s: %w", event, err)
	}
	// Watchdog: if a wsCollected never closes this (the spec failed between the
	// two), tear the connection down after a bounded lifetime so the read loop
	// cannot outlive the test. wsCollected closing the sub first ends the read
	// loop; this only backstops the failure path.
	watchCtx, cancel := context.WithTimeout(ctx, wsCaptureMaxLifetime)
	go func() {
		defer cancel()
		<-watchCtx.Done()
		_ = sub.Close()
	}()
	ac.Value = sub
	return nil
}

// logsFilter builds an eth_subscribe "logs" filter object from an action's
// address and topics arguments, or nil when neither is given (subscribe to all
// logs). topics passes through verbatim so a spec can use the null wildcard.
func logsFilter(d *interp.Deps, args map[string]any) (map[string]any, error) {
	filter := map[string]any{}
	if ref, ok := args["address"].(string); ok && ref != "" {
		addr, err := ResolveAddress(d, ref)
		if err != nil {
			return nil, err
		}
		filter["address"] = addr
	}
	if topics, ok := args["topics"].([]any); ok {
		filter["topics"] = topics
	}
	if len(filter) == 0 {
		return nil, nil
	}
	return filter, nil
}

// wsCollectedAssertion drains a subscription an earlier wsOpen bound, passing
// when at least "count" valid notifications arrived within the timeout. Args:
// sub (the "$handle" an earlier wsOpen saved), count (default 1), timeout, on.
type wsCollectedAssertion struct{}

func (wsCollectedAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertWSCollected, Provenance: sanitizeWSProvenance(ac.Spec)}
	sub, ok := ac.Spec["sub"].(*rpc.Subscription)
	if !ok {
		err := fmt.Errorf("dsl: wsCollected requires \"sub\" — the handle an earlier wsOpen saved")
		res.Actual = err.Error()
		return res, err
	}
	defer func() { _ = sub.Close() }()
	want := 1
	if n, ok := uintArg(ac.Spec["count"]); ok && n > 0 {
		want = int(n)
	}
	res.Expected = strconv.Itoa(want)
	timeout := durationArg(ac.Spec, "timeout", defaultSubscribeTimeout)
	sctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

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
			res.Source = fmt.Sprintf("only %d of %d notification(s) within %s", got, want, timeout)
			return res, nil
		}
	}
	res.Actual = strconv.Itoa(got)
	res.Pass = true
	return res, nil
}

// sanitizeWSProvenance copies an assertion's spec for the record without the
// live subscription handle, which is a channel-backed struct that cannot be
// marshaled to JSON. The handle's id stands in its place.
func sanitizeWSProvenance(spec map[string]any) map[string]any {
	out := make(map[string]any, len(spec))
	for k, v := range spec {
		if sub, ok := v.(*rpc.Subscription); ok {
			out[k] = "subscription " + sub.ID()
			continue
		}
		out[k] = v
	}
	return out
}

// readGasPrice returns eth_gasPrice as a decimal string. Together with baseFee
// it covers the gas-policy cases: on an EIP-1559 chain the suggested price is
// the base fee plus the node's tip, so comparing the two is how a spec checks
// the tip without the harness having to know the chain's tip rule.

// A timeout is a failed assertion reporting the count that did arrive, not an
// error — "two heads in five seconds" is a claim that can simply be false.
type wsSubscribeAssertion struct{}

func (wsSubscribeAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
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
	spec, rerr := resolveAddressArgs(ac.Deps, ac.Spec)
	if rerr != nil {
		res.Pass, res.Actual = false, rerr.Error()
		return res, rerr
	}
	params := []any{event}
	if extra, ok := spec["params"].([]any); ok {
		params = append(params, extra...)
	}
	want := 1
	if n, ok := uintArg(ac.Spec["count"]); ok && n > 0 {
		want = int(n)
	}
	res.Expected = spec["expected"]
	if res.Expected == nil {
		res.Expected = strconv.Itoa(want)
	}

	timeout := durationArg(ac.Spec, "timeout", defaultSubscribeTimeout)
	sctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sub, err := rpc.Subscribe(sctx, wsURL, params...)
	if err != nil {
		res.Actual = err.Error()
		return res, fmt.Errorf("dsl: wsSubscribe %s: %w", event, err)
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

// wsTarget derives the WebSocket URL of the assertion's target node.
func wsTarget(ac *interp.AssertCtx) (string, error) {
	nodes := ac.On
	if len(nodes) == 0 && ac.Env != nil {
		nodes = ac.Env.Nodes()
	}
	return wsTargetURL(nodes)
}

// nodesFor resolves an action's target nodes: the "on" selector if present,
// else the environment's node table.
func nodesFor(env interp.NodeTable, args map[string]any) []node.Node {
	if env == nil {
		return nil
	}
	if sel, ok := args["on"].(string); ok && sel != "" {
		if n, err := env.Resolve(sel); err == nil {
			return []node.Node{n}
		}
		return nil
	}
	return env.Nodes()
}

// wsTargetURL builds the WebSocket URL of the first node from its host and WS
// port. Attached nodes carry no port map, so this names that rather than dialing
// something wrong.
func wsTargetURL(nodes []node.Node) (string, error) {
	if len(nodes) == 0 {
		return "", fmt.Errorf("dsl: no target node for a WebSocket subscription")
	}
	n := nodes[0]
	// The recorded endpoint wins. Host and Ports hold the node's OWN address,
	// which under docker is the container-internal one this tool cannot route
	// to; the composition already resolved the reachable form through the same
	// opener every HTTP dial uses. Building from Host+Ports is what made this
	// the one dial that skipped the translation.
	if n.WSURL != "" {
		return n.WSURL, nil
	}
	if n.Ports.WS == 0 {
		return "", fmt.Errorf("dsl: node%d has no WebSocket port (an attached node's ports are unknown)", n.Index)
	}
	host := n.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("ws://%s:%d", host, n.Ports.WS), nil
}

// readDerive computes an arithmetic combination of already-bound values —
// the saved-value arithmetic the migration ledger recorded as a gap
// (gasPrice == baseFee + gasTip; block-period timestamp diffs). Spec:
// op (sum | diff), of (values or "$bindings"; hex 0x or decimal strings, or
// numbers). diff subtracts the rest from the first. The result is a decimal
// string, comparable with the numeric assert primitives.
//
// format: "hex" returns a 0x-hex quantity instead. A block-scoped RPC method
// takes its block as 0x-hex and rejects a decimal string, so without it a spec
// could read a receipt's block number but not ask anything about the block
// after it — which is the only way to check a rule stated over a block and its
// child, such as the base fee an in-band block hands on unchanged.
