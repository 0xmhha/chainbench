package testspec

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testspec/assert"
)

// Built-in action and assertion names (the DSL keys the interpreter dispatches
// on). Kept as typed-free string constants so the registry and specs share one
// source of truth.
const (
	actionSendTx    = "sendTx"
	actionWaitBlock = "waitBlock"

	assertChainID     = "chainId"
	assertBlockNumber = "blockNumber"
	assertPeerCount   = "peerCount"
	assertBalanceAt   = "balanceAt"
	assertCodeAt      = "codeAt"
	assertNonceAt     = "nonceAt"
	assertCall        = "call"
	assertTxStatus    = "txStatus"
)

// Defaults for the sendTx wait loop, overridable per action via args.
const (
	defaultTxTimeout      = 30 * time.Second
	defaultTxPollInterval = 500 * time.Millisecond
)

// Defaults for the waitBlock poll loop, overridable per action via args.
const (
	defaultWaitBlockTimeout = 60 * time.Second
	defaultWaitBlockPoll    = 500 * time.Millisecond
)

// seedBuiltins registers the built-in tx action and RPC-reading assertions on r.
// It is called by NewRegistry(true).
func seedBuiltins(r Registry) {
	r.RegisterAction(actionSendTx, sendTxAction{})
	r.RegisterAction(actionWaitBlock, waitBlockAction{})
	for _, a := range builtinAssertions() {
		r.RegisterAssertion(a.name, a)
	}
}

// waitBlockAction blocks until the target node's height reaches "target" (a
// number) or the timeout elapses. Args: target, on, timeout, pollInterval.
type waitBlockAction struct{}

func (waitBlockAction) Do(ctx context.Context, ac *ActionCtx) error {
	target, ok := uintArg(ac.Args["target"])
	if !ok {
		return fmt.Errorf("testspec: waitBlock requires a numeric \"target\"")
	}
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, durationArg(ac.Args, "timeout", defaultWaitBlockTimeout))
	defer cancel()
	t := time.NewTicker(durationArg(ac.Args, "pollInterval", defaultWaitBlockPoll))
	defer t.Stop()
	for {
		if bn, err := c.BlockNumber(ctx); err == nil && bn >= target {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("testspec: waitBlock: height %d not reached: %w", target, ctx.Err())
		case <-t.C:
		}
	}
}

// builtinAssertions lists the RPC-reading assertions and their default
// comparator. Each reads one value from a target node and compares it to the
// spec's "expected" using an assert primitive ("compare" in the spec overrides
// the default).
func builtinAssertions() []rpcAssertion {
	return []rpcAssertion{
		{name: assertChainID, defaultOp: "Equal", read: readChainID},
		{name: assertBlockNumber, defaultOp: "GreaterOrEqual", read: readBlockNumber},
		{name: assertPeerCount, defaultOp: "GreaterOrEqual", read: readPeerCount},
		{name: assertBalanceAt, defaultOp: "Equal", read: readBalanceAt},
		{name: assertCodeAt, defaultOp: "Equal", read: readCodeAt},
		{name: assertNonceAt, defaultOp: "Equal", read: readNonceAt},
		{name: assertCall, defaultOp: "Equal", read: readCall},
		{name: assertTxStatus, defaultOp: "Equal", read: readTxStatus},
	}
}

// sendTxAction submits a node-signed transaction and, unless wait:false, polls
// for its receipt before returning.
type sendTxAction struct{}

// Do resolves the target node, sends the transaction, and waits for the
// receipt. Args: on (selector, default primary), from, to, value, gas, wait
// (default true), timeout, pollInterval.
func (sendTxAction) Do(ctx context.Context, ac *ActionCtx) error {
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	from, _ := ac.Args["from"].(string)
	if from == "" {
		return fmt.Errorf("testspec: sendTx requires \"from\"")
	}
	args := rpc.SendTxArgs{From: from}
	if to, ok := ac.Args["to"].(string); ok {
		args.To = to
	}
	if data, ok := ac.Args["data"].(string); ok {
		args.Data = data
	}
	if v, ok := hexQuantity(ac.Args["value"]); ok {
		args.Value = v
	}
	if g, ok := hexQuantity(ac.Args["gas"]); ok {
		args.Gas = g
	}
	hash, err := c.SendTransaction(ctx, args)
	if err != nil {
		return fmt.Errorf("testspec: sendTx: %w", err)
	}
	if wait, ok := ac.Args["wait"].(bool); ok && !wait {
		return nil
	}
	return waitReceipt(ctx, c, hash,
		durationArg(ac.Args, "timeout", defaultTxTimeout),
		durationArg(ac.Args, "pollInterval", defaultTxPollInterval))
}

// waitReceipt polls for a transaction receipt until it appears or ctx/timeout
// expires. It probes immediately, so a mined transaction returns without any
// sleep.
func waitReceipt(ctx context.Context, c *rpc.Client, hash string, timeout, interval time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		if raw, err := c.TxReceipt(ctx, hash); err == nil && raw != nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("testspec: sendTx: receipt %s: %w", hash, ctx.Err())
		case <-t.C:
		}
	}
}

// reader reads one value from a node for an assertion. The value is returned in
// a form the assert primitives compare (decimal string or 0x-hex).
type reader func(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error)

// rpcAssertion compares one RPC-read value to the spec's expected value.
type rpcAssertion struct {
	name      string
	defaultOp string
	read      reader
}

// Check reads the value from the target node and compares it to "expected"
// using the default comparator (or the spec's "compare" override).
func (a rpcAssertion) Check(ctx context.Context, ac *AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: a.name, Provenance: ac.Spec}
	c, err := clientFor(ac.Deps, targetURL(ac))
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	actual, err := a.read(ctx, c, ac.Spec)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	op := a.defaultOp
	if o, ok := ac.Spec["compare"].(string); ok && o != "" {
		op = o
	}
	fn, ok := assert.Lookup(op)
	if !ok {
		return res, fmt.Errorf("testspec: unknown comparator %q", op)
	}
	expected := ac.Spec["expected"]
	pass, detail := fn(actual, expected)
	res.Expected = expected
	res.Actual = actual
	res.Pass = pass
	if !pass && detail != "" {
		res.Source = detail
	}
	return res, nil
}

func readChainID(ctx context.Context, c *rpc.Client, _ map[string]any) (any, error) {
	v, err := c.ChainID(ctx)
	return strconv.FormatUint(v, 10), err
}

func readBlockNumber(ctx context.Context, c *rpc.Client, _ map[string]any) (any, error) {
	v, err := c.BlockNumber(ctx)
	return strconv.FormatUint(v, 10), err
}

func readPeerCount(ctx context.Context, c *rpc.Client, _ map[string]any) (any, error) {
	v, err := c.PeerCount(ctx)
	return strconv.FormatUint(v, 10), err
}

func readBalanceAt(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	addr, ok := spec["address"].(string)
	if !ok || addr == "" {
		return nil, fmt.Errorf("testspec: balanceAt requires \"address\"")
	}
	v, err := c.BalanceAt(ctx, addr)
	if err != nil {
		return nil, err
	}
	return bigString(v), nil
}

func readCodeAt(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	addr, ok := spec["address"].(string)
	if !ok || addr == "" {
		return nil, fmt.Errorf("testspec: codeAt requires \"address\"")
	}
	return c.CodeAt(ctx, addr)
}

func readNonceAt(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	addr, ok := spec["address"].(string)
	if !ok || addr == "" {
		return nil, fmt.Errorf("testspec: nonceAt requires \"address\"")
	}
	v, err := c.NonceAt(ctx, addr)
	if err != nil {
		return nil, err
	}
	return strconv.FormatUint(v, 10), nil
}

// readCall runs a read-only contract call (eth_call) and returns the 0x-hex
// result, for asserting on-chain state (e.g. a governance getter).
func readCall(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	to, ok := spec["to"].(string)
	if !ok || to == "" {
		return nil, fmt.Errorf("testspec: call requires \"to\"")
	}
	data, ok := spec["data"].(string)
	if !ok || data == "" {
		return nil, fmt.Errorf("testspec: call requires \"data\"")
	}
	return c.EthCall(ctx, to, data)
}

// readTxStatus returns a transaction receipt's status ("0x1" success, "0x0"
// reverted), for asserting positive and negative (expectRevert) tx outcomes.
func readTxStatus(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	hash, ok := spec["hash"].(string)
	if !ok || hash == "" {
		return nil, fmt.Errorf("testspec: txStatus requires \"hash\"")
	}
	raw, err := c.TxReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("testspec: txStatus: no receipt for %s", hash)
	}
	var r struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("testspec: txStatus: parse receipt: %w", err)
	}
	return r.Status, nil
}

// clientFor returns an RPC client for url, guarding a missing injected factory.
func clientFor(deps *Deps, url string) (*rpc.Client, error) {
	if deps == nil || deps.RPC == nil {
		return nil, fmt.Errorf("testspec: no RPC client injected")
	}
	if url == "" {
		return nil, fmt.Errorf("testspec: no target node RPC URL")
	}
	return deps.RPC(url), nil
}

// targetURL is the assertion's target node URL: the first resolved "on" node,
// else the environment's primary node.
func targetURL(ac *AssertCtx) string {
	if len(ac.On) > 0 {
		return ac.On[0].RPCURL
	}
	if ac.Env != nil {
		if nodes := ac.Env.Nodes(); len(nodes) > 0 {
			return nodes[0].RPCURL
		}
	}
	return ""
}

// selectorTarget resolves an action's "on" selector to a node URL, else the
// environment's primary node.
func selectorTarget(env session.Environment, args map[string]any) string {
	if env == nil {
		return ""
	}
	if sel, ok := args["on"].(string); ok && sel != "" {
		if n, err := env.Resolve(sel); err == nil {
			return n.RPCURL
		}
		return ""
	}
	if nodes := env.Nodes(); len(nodes) > 0 {
		return nodes[0].RPCURL
	}
	return ""
}

// hexQuantity normalizes a decimal/0x-hex/number value to a 0x-hex quantity.
// ok is false when v is absent or unparseable.
func hexQuantity(v any) (string, bool) {
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return "", false
		}
		if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
			return s, true
		}
		bi, ok := new(big.Int).SetString(s, 10)
		if !ok {
			return "", false
		}
		return "0x" + bi.Text(16), true
	case float64:
		bi := new(big.Int)
		big.NewFloat(x).Int(bi)
		return "0x" + bi.Text(16), true
	case int:
		return "0x" + strconv.FormatInt(int64(x), 16), true
	default:
		return "", false
	}
}

// bigString renders a big.Int as a decimal string ("0" for nil).
func bigString(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}

// uintArg normalizes a numeric arg (number or decimal string) to a uint64.
// ok is false when v is absent, negative, or unparseable.
func uintArg(v any) (uint64, bool) {
	switch x := v.(type) {
	case float64:
		if x < 0 {
			return 0, false
		}
		return uint64(x), true
	case int:
		if x < 0 {
			return 0, false
		}
		return uint64(x), true
	case string:
		n, err := strconv.ParseUint(strings.TrimSpace(x), 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

// durationArg reads a Go duration string arg, falling back to def.
func durationArg(args map[string]any, key string, def time.Duration) time.Duration {
	if s, ok := args[key].(string); ok && s != "" {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return def
}
