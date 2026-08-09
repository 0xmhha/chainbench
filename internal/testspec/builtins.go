package testspec

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
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
	actionRead      = "read"

	assertChainID       = "chainId"
	assertBlockNumber   = "blockNumber"
	assertPeerCount     = "peerCount"
	assertBalanceAt     = "balanceAt"
	assertCodeAt        = "codeAt"
	assertNonceAt       = "nonceAt"
	assertCall          = "call"
	assertTxStatus      = "txStatus"
	assertBlockAdvance  = "blockAdvance"
	assertSameBlockHash = "sameBlockHash"
	assertBaseFee       = "baseFee"
	assertEstimateGas   = "estimateGas"
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

// Defaults for the blockAdvance assertion poll loop, overridable per spec.
const (
	defaultBlockAdvanceTimeout = 30 * time.Second
	defaultBlockAdvancePoll    = 500 * time.Millisecond
)

// seedBuiltins registers the built-in tx action and RPC-reading assertions on r.
// It is called by NewRegistry(true).
func seedBuiltins(r Registry) {
	r.RegisterAction(actionSendTx, sendTxAction{})
	r.RegisterAction(actionWaitBlock, waitBlockAction{})
	r.RegisterAction(actionRead, readAction{})
	seedFaultBuiltins(r)
	r.RegisterAssertion(assertBlockAdvance, blockAdvanceAssertion{})
	r.RegisterAssertion(assertSameBlockHash, sameBlockHashAssertion{})
	for _, a := range builtinAssertions() {
		r.RegisterAssertion(a.name, a)
	}
}

// sameBlockHashAssertion passes when every target node reports the same hash for
// a block — a cross-node no-fork / same-chain check. Spec: block (tag, default
// "latest"; use "0x0" for genesis agreement), onEach. Non-answering nodes are
// skipped; it fails if no node answers.
type sameBlockHashAssertion struct{}

func (sameBlockHashAssertion) Check(ctx context.Context, ac *AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertSameBlockHash, Provenance: ac.Spec}
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("testspec: sameBlockHash: no target node RPC URL")
		res.Actual = err.Error()
		return res, err
	}
	tag, _ := ac.Spec["block"].(string)
	if tag == "" {
		tag = "latest"
	}
	res.Expected = "all nodes agree on block " + tag + " hash"

	var hashes []string
	perNode := make(map[string]any, len(targets))
	for _, tgt := range targets {
		c, err := clientFor(ac.Deps, tgt.url)
		if err != nil {
			res.Actual = err.Error()
			return res, err
		}
		b, err := c.BlockByNumber(ctx, tag)
		if err != nil {
			res.Actual = err.Error()
			return res, err
		}
		if b.Hash == "" {
			continue // node has not produced this block yet
		}
		hashes = append(hashes, b.Hash)
		perNode[tgt.name] = b.Hash
	}
	res.Actual = perNode
	if len(hashes) == 0 {
		res.Pass, res.Source = false, "no node returned block "+tag
		return res, nil
	}
	pass, detail := assert.HashesEqual(hashes)
	res.Pass = pass
	if !pass {
		res.Source = detail
	}
	return res, nil
}

// blockAdvanceAssertion passes when the target node's head advances within the
// poll window — proof the network is producing blocks. Spec: timeout,
// pollInterval, on. It reads the head once, then polls until a higher head or
// the timeout.
type blockAdvanceAssertion struct{}

func (blockAdvanceAssertion) Check(ctx context.Context, ac *AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertBlockAdvance, Provenance: ac.Spec}
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("testspec: blockAdvance: no target node RPC URL")
		res.Actual = err.Error()
		return res, err
	}
	c, err := clientFor(ac.Deps, targets[0].url)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	start, err := c.BlockNumber(ctx)
	if err != nil {
		res.Actual = err.Error()
		return res, err
	}
	res.Expected = "head > " + strconv.FormatUint(start, 10)

	timeout := durationArg(ac.Spec, "timeout", defaultBlockAdvanceTimeout)
	pctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	t := time.NewTicker(durationArg(ac.Spec, "pollInterval", defaultBlockAdvancePoll))
	defer t.Stop()
	for {
		if cur, err := c.BlockNumber(pctx); err == nil && cur > start {
			res.Pass, res.Actual = true, cur
			return res, nil
		}
		select {
		case <-pctx.Done():
			res.Pass, res.Actual = false, start
			res.Source = "head did not advance within " + timeout.String()
			return res, nil
		case <-t.C:
		}
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

// readAction reads one RPC value and, with "save", binds it for later steps and
// assertions to reference as "$name" (design §3.2b). It is how a spec compares
// two on-chain reads to each other — read the first, then assert the second
// against "$name" — which a single-shot assertion cannot express.
//
// Args: source (one of the RPC-reading assertion names), save, on, plus whatever
// that source needs (to/data for "call", address for "balanceAt", ...).
type readAction struct{}

func (readAction) Do(ctx context.Context, ac *ActionCtx) error {
	source, _ := ac.Args["source"].(string)
	if source == "" {
		return fmt.Errorf("testspec: read requires a \"source\" (one of: %s)", strings.Join(readerNames(), ", "))
	}
	read, ok := readerFor(source)
	if !ok {
		return fmt.Errorf("testspec: read: unknown source %q (one of: %s)", source, strings.Join(readerNames(), ", "))
	}
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	v, err := read(ctx, c, ac.Args)
	if err != nil {
		return fmt.Errorf("testspec: read %s: %w", source, err)
	}
	ac.Value = v
	return nil
}

// readerFor returns the reader registered under an assertion name, so the read
// action and the assertions share one vocabulary (no second spelling of "call").
func readerFor(name string) (reader, bool) {
	for _, a := range builtinAssertions() {
		if a.name == name {
			return a.read, true
		}
	}
	return nil, false
}

// readerNames lists the sources the read action accepts, for error messages.
func readerNames() []string {
	all := builtinAssertions()
	out := make([]string, 0, len(all))
	for _, a := range all {
		out = append(out, a.name)
	}
	sort.Strings(out)
	return out
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
		{name: assertBaseFee, defaultOp: "GreaterOrEqual", read: readBaseFee},
		{name: assertEstimateGas, defaultOp: "GreaterOrEqual", read: readEstimateGas},
	}
}

// readEstimateGas returns eth_estimateGas for a call as a decimal string, for
// gas-cost bounds (e.g. a contract call exceeds the 21000 bare-transfer floor).
func readEstimateGas(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	to, _ := spec["to"].(string)
	if to == "" {
		return nil, fmt.Errorf("testspec: estimateGas requires \"to\"")
	}
	data, _ := spec["data"].(string)
	if data == "" {
		return nil, fmt.Errorf("testspec: estimateGas requires \"data\"")
	}
	from, _ := spec["from"].(string)
	g, err := c.EstimateGas(ctx, from, to, data)
	if err != nil {
		return nil, err
	}
	return strconv.FormatUint(g, 10), nil
}

// readBaseFee returns the latest block's baseFeePerGas as a decimal string, for
// gas-policy bounds checks. It errors on a pre-EIP-1559 block (no base fee).
func readBaseFee(ctx context.Context, c *rpc.Client, _ map[string]any) (any, error) {
	b, err := c.BlockByNumber(ctx, "latest")
	if err != nil {
		return nil, err
	}
	if b.BaseFeePerGas == nil {
		return nil, fmt.Errorf("testspec: baseFee: block has no baseFeePerGas")
	}
	return b.BaseFeePerGas.String(), nil
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
	ac.Hash = hash
	if wait, ok := ac.Args["wait"].(bool); ok && !wait {
		return nil
	}
	receipt, err := waitReceipt(ctx, c, hash,
		durationArg(ac.Args, "timeout", defaultTxTimeout),
		durationArg(ac.Args, "pollInterval", defaultTxPollInterval))
	if err != nil {
		return err
	}
	ac.Receipt = receipt
	return checkTxOutcome(hash, receipt, ac.Args)
}

// checkTxOutcome enforces a tx step's declared expectation (F11 — a tx step is
// atomically successful only when its expectation is met). By default the
// transaction must not revert. With expectRevert (or expect:"revert") the
// transaction MUST revert: a success then fails the step, so negative cases are
// expressed declaratively.
func checkTxOutcome(hash string, receipt map[string]any, args map[string]any) error {
	reverted := statusReverted(receipt)
	if wantRevert(args) {
		if !reverted {
			return fmt.Errorf("testspec: sendTx %s expected revert but succeeded", hash)
		}
		return nil
	}
	if reverted {
		return fmt.Errorf("testspec: sendTx %s reverted (status 0x0)", hash)
	}
	return nil
}

// wantRevert reports whether a tx step declares that it expects a revert, via
// expectRevert:true or expect:"revert".
func wantRevert(args map[string]any) bool {
	if b, ok := args["expectRevert"].(bool); ok {
		return b
	}
	if s, ok := args["expect"].(string); ok {
		return strings.EqualFold(s, "revert")
	}
	return false
}

// statusReverted reports whether a receipt's status is an explicit revert (0x0).
// A missing status (legacy pre-Byzantium receipts) is treated as success.
func statusReverted(receipt map[string]any) bool {
	status, _ := receipt["status"].(string)
	return status == "0x0" || status == "0x00"
}

// waitReceipt polls for a transaction receipt until it appears or ctx/timeout
// expires, returning the parsed receipt. It probes immediately, so a mined
// transaction returns without any sleep.
func waitReceipt(ctx context.Context, c *rpc.Client, hash string, timeout, interval time.Duration) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		if raw, err := c.TxReceipt(ctx, hash); err == nil && raw != nil {
			var m map[string]any
			if uerr := json.Unmarshal(raw, &m); uerr != nil {
				return nil, fmt.Errorf("testspec: sendTx: parse receipt %s: %w", hash, uerr)
			}
			return m, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("testspec: sendTx: receipt %s: %w", hash, ctx.Err())
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
	res := session.AssertResult{Assert: a.name, Provenance: ac.Spec, Pass: true}
	op := a.defaultOp
	if o, ok := ac.Spec["compare"].(string); ok && o != "" {
		op = o
	}
	fn, ok := assert.Lookup(op)
	if !ok {
		return res, fmt.Errorf("testspec: unknown comparator %q", op)
	}
	expected := ac.Spec["expected"]
	res.Expected = expected

	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("testspec: no target node RPC URL")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}

	// Read and compare on every target node ("on" is one; "onEach" is many). All
	// must satisfy the comparator; a failure names the node.
	actuals := make(map[string]any, len(targets))
	for _, tgt := range targets {
		c, err := clientFor(ac.Deps, tgt.url)
		if err != nil {
			res.Pass, res.Actual = false, err.Error()
			return res, err
		}
		actual, err := a.read(ctx, c, ac.Spec)
		if err != nil {
			res.Pass, res.Actual = false, err.Error()
			return res, err
		}
		actuals[tgt.name] = actual
		if pass, detail := fn(actual, expected); !pass {
			res.Pass = false
			if detail == "" {
				detail = "mismatch"
			}
			res.Source = tgt.name + ": " + detail
		}
	}
	if len(targets) == 1 {
		res.Actual = actuals[targets[0].name] // scalar for the common single-node case
	} else {
		res.Actual = actuals
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

// assertTarget is one node an assertion reads from.
type assertTarget struct {
	name string
	url  string
}

// assertTargets are the nodes an assertion checks: every resolved "on"/"onEach"
// node, else the environment's primary node.
func assertTargets(ac *AssertCtx) []assertTarget {
	if len(ac.On) > 0 {
		out := make([]assertTarget, 0, len(ac.On))
		for _, n := range ac.On {
			out = append(out, assertTarget{name: "node" + strconv.Itoa(n.Index), url: n.RPCURL})
		}
		return out
	}
	if ac.Env != nil {
		if nodes := ac.Env.Nodes(); len(nodes) > 0 {
			return []assertTarget{{name: "node" + strconv.Itoa(nodes[0].Index), url: nodes[0].RPCURL}}
		}
	}
	return nil
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
