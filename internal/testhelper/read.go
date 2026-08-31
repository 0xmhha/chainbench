package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/assert"
)

// sameBlockHashAssertion passes when every target node reports the same hash for
// a block — a cross-node no-fork / same-chain check. Spec: block (tag, default
// "latest"; use "0x0" for genesis agreement), onEach. Non-answering nodes are
// skipped; it fails if no node answers.
type sameBlockHashAssertion struct{}

func (sameBlockHashAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertSameBlockHash, Provenance: ac.Spec}
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: sameBlockHash: no target node RPC URL")
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

func (blockAdvanceAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertBlockAdvance, Provenance: ac.Spec}
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: blockAdvance: no target node RPC URL")
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

// readAction reads one RPC value and, with "save", binds it for later steps and
// assertions to reference as "$name" (design §3.2b). It is how a spec compares
// two on-chain reads to each other — read the first, then assert the second
// against "$name" — which a single-shot assertion cannot express.
//
// Args: source (one of the RPC-reading assertion names), save, on, plus whatever
// that source needs (to/data for "call", address for "balanceAt", ...).
type readAction struct{}

func (readAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	source, _ := ac.Args["source"].(string)
	if source == "" {
		return fmt.Errorf("dsl: read requires a \"source\" (one of: %s)", strings.Join(readerNames(), ", "))
	}
	read, ok := ac.Deps.Actions.Reader(source)
	if !ok {
		return fmt.Errorf("dsl: read: unknown source %q (one of: %s)", source, strings.Join(readerNames(), ", "))
	}
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	v, err := read(ctx, c, ac.Args)
	if err != nil {
		return fmt.Errorf("dsl: read %s: %w", source, err)
	}
	ac.Value = v
	return nil
}

// Defaults for the waitFor poll loop, overridable per action via args.
const (
	defaultWaitForTimeout = 30 * time.Second
	defaultWaitForPoll    = 500 * time.Millisecond
)

// waitForAction polls a read source until its value satisfies a comparator, or
// the timeout elapses. It composes the reader vocabulary (the same "source"
// names as read/assert) with the assert comparators, so a spec can wait on any
// readable value — a header field governance updates, a proposal status — not
// just block height (waitBlock is the special case source:blockNumber
// compare:GreaterOrEqual). Args: source (required), compare (default "Equal"),
// expected (required), on, timeout, pollInterval, plus the source's own args.
// The satisfying value is bound under "save" like a read.
type waitForAction struct{}

func (waitForAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	source, _ := ac.Args["source"].(string)
	if source == "" {
		return fmt.Errorf("dsl: waitFor requires a \"source\" (one of: %s)", strings.Join(readerNames(), ", "))
	}
	read, ok := ac.Deps.Actions.Reader(source)
	if !ok {
		return fmt.Errorf("dsl: waitFor: unknown source %q (one of: %s)", source, strings.Join(readerNames(), ", "))
	}
	op, _ := ac.Args["compare"].(string)
	if op == "" {
		op = "Equal"
	}
	cmp, ok := assert.Lookup(op)
	if !ok {
		return fmt.Errorf("dsl: waitFor: unknown comparator %q", op)
	}
	expected := ac.Args["expected"]
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, durationArg(ac.Args, "timeout", defaultWaitForTimeout))
	defer cancel()
	t := time.NewTicker(durationArg(ac.Args, "pollInterval", defaultWaitForPoll))
	defer t.Stop()
	var lastActual any
	var lastErr error
	for {
		if v, rerr := read(ctx, c, ac.Args); rerr == nil {
			lastActual, lastErr = v, nil
			if pass, _ := cmp(v, expected); pass {
				ac.Value = v
				return nil
			}
		} else {
			lastErr = rerr
		}
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("dsl: waitFor %s: condition unmet (last read error: %v): %w", source, lastErr, ctx.Err())
			}
			return fmt.Errorf("dsl: waitFor %s: %v did not satisfy %s %v within timeout: %w", source, lastActual, op, expected, ctx.Err())
		case <-t.C:
		}
	}
}

// readerNames lists the sources the built-ins register, for error messages.
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
		{name: assertReceiptLog, defaultOp: "Equal", read: readReceiptLog},
		{name: assertBaseFee, defaultOp: "GreaterOrEqual", read: readBaseFee},
		{name: assertEstimateGas, defaultOp: "GreaterOrEqual", read: readEstimateGas},
		{name: assertLogs, defaultOp: "Equal", read: readLogs},
		{name: assertGasPrice, defaultOp: "GreaterOrEqual", read: readGasPrice},
		{name: assertRPCCall, defaultOp: "Equal", read: readRPCCall},
		{name: assertDerive, defaultOp: "Equal", read: readDerive},
	}
}

// reader reads one value from a node for an assertion. The value is returned in
// a form the assert primitives compare (decimal string or 0x-hex).
type reader = interp.Reader

// rpcAssertion compares one RPC-read value to the spec's expected value.
type rpcAssertion struct {
	name      string
	defaultOp string
	read      reader
}

// Check reads the value from the target node and compares it to "expected"
// using the default comparator (or the spec's "compare" override).
func (a rpcAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: a.name, Provenance: ac.Spec, Pass: true}
	op := a.defaultOp
	if o, ok := ac.Spec["compare"].(string); ok && o != "" {
		op = o
	}
	fn, ok := assert.Lookup(op)
	if !ok {
		return res, fmt.Errorf("dsl: unknown comparator %q", op)
	}
	expected := ac.Spec["expected"]
	res.Expected = expected

	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: no target node RPC URL")
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

// readEstimateGas returns eth_estimateGas for a call as a decimal string, for
// gas-cost bounds (e.g. a contract call exceeds the 21000 bare-transfer floor).
func readEstimateGas(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	to, _ := spec["to"].(string)
	if to == "" {
		return nil, fmt.Errorf("dsl: estimateGas requires \"to\"")
	}
	data, _ := spec["data"].(string)
	if data == "" {
		return nil, fmt.Errorf("dsl: estimateGas requires \"data\"")
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
		return nil, fmt.Errorf("dsl: baseFee: block has no baseFeePerGas")
	}
	return b.BaseFeePerGas.String(), nil
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
		return nil, fmt.Errorf("dsl: balanceAt requires \"address\"")
	}
	v, err := c.BalanceAt(ctx, addr)
	if err != nil {
		return nil, err
	}
	return interp.BigString(v), nil
}

func readCodeAt(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	addr, ok := spec["address"].(string)
	if !ok || addr == "" {
		return nil, fmt.Errorf("dsl: codeAt requires \"address\"")
	}
	return c.CodeAt(ctx, addr)
}

func readNonceAt(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	addr, ok := spec["address"].(string)
	if !ok || addr == "" {
		return nil, fmt.Errorf("dsl: nonceAt requires \"address\"")
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
		return nil, fmt.Errorf("dsl: call requires \"to\"")
	}
	data, ok := spec["data"].(string)
	if !ok || data == "" {
		return nil, fmt.Errorf("dsl: call requires \"data\"")
	}
	return c.EthCall(ctx, to, data)
}

// readTxStatus returns a transaction receipt's status ("0x1" success, "0x0"
// reverted), for asserting positive and negative (expectRevert) tx outcomes.
func readTxStatus(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	hash, ok := spec["hash"].(string)
	if !ok || hash == "" {
		return nil, fmt.Errorf("dsl: txStatus requires \"hash\"")
	}
	raw, err := c.TxReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("dsl: txStatus: no receipt for %s", hash)
	}
	var r struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("dsl: txStatus: parse receipt: %w", err)
	}
	return r.Status, nil
}

// readReceiptLog extracts one field from a transaction receipt's event logs so a
// value emitted at run time — the classic case is a governance proposalId, an
// indexed topic on a ProposalCreated log — can be bound and spliced into later
// calldata (via derive abiCall). Spec: hash (required); address / topic0
// (optional filters, hex-case-insensitive); index (which matching log, default
// 0); topic (which topic to return, default 1) or select:"data" for the log
// data. The returned 32-byte 0x-hex is directly usable as an abiCall argument.
func readReceiptLog(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	hash, ok := spec["hash"].(string)
	if !ok || hash == "" {
		return nil, fmt.Errorf("dsl: receiptLog requires \"hash\"")
	}
	raw, err := c.TxReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}
	if raw == nil {
		return nil, fmt.Errorf("dsl: receiptLog: no receipt for %s", hash)
	}
	var r struct {
		Logs []struct {
			Address string   `json:"address"`
			Topics  []string `json:"topics"`
			Data    string   `json:"data"`
		} `json:"logs"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("dsl: receiptLog: parse receipt: %w", err)
	}

	wantAddr, _ := spec["address"].(string)
	wantTopic0, _ := spec["topic0"].(string)
	idx := 0
	if n, ok := uintArg(spec["index"]); ok {
		idx = int(n)
	}
	seen := 0
	for _, lg := range r.Logs {
		if wantAddr != "" && !strings.EqualFold(lg.Address, wantAddr) {
			continue
		}
		if wantTopic0 != "" && (len(lg.Topics) == 0 || !strings.EqualFold(lg.Topics[0], wantTopic0)) {
			continue
		}
		if seen != idx {
			seen++
			continue
		}
		if sel, _ := spec["select"].(string); sel == "data" {
			return lg.Data, nil
		}
		topic := 1
		if n, ok := uintArg(spec["topic"]); ok {
			topic = int(n)
		}
		if topic < 0 || topic >= len(lg.Topics) {
			return nil, fmt.Errorf("dsl: receiptLog: log has no topic %d (has %d)", topic, len(lg.Topics))
		}
		return lg.Topics[topic], nil
	}
	return nil, fmt.Errorf("dsl: receiptLog: no matching log in receipt for %s", hash)
}
