package testhelper

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// Built-in action and assertion names (the DSL keys the interpreter dispatches
// on). Kept as typed-free string constants so the registry and specs share one
// source of truth.
const (
	actionSendTx          = "sendTx"
	actionWaitBlock       = "waitBlock"
	actionWaitFor         = "waitFor"
	actionNewAccount      = "newAccount"
	actionSendRawTampered = "sendRawTampered"

	assertChainID       = "chainId"
	assertBlockNumber   = "blockNumber"
	assertPeerCount     = "peerCount"
	assertBalanceAt     = "balanceAt"
	assertCodeAt        = "codeAt"
	assertNonceAt       = "nonceAt"
	assertCall          = "call"
	assertTxStatus      = "txStatus"
	assertReceiptLog    = "receiptLog"
	assertBlockAdvance  = "blockAdvance"
	assertSameBlockHash = "sameBlockHash"
	assertBaseFee       = "baseFee"
	assertEstimateGas   = "estimateGas"
	assertTxMined       = "txMined"
	assertCallError     = "callError"
	assertMethodPresent = "methodPresent"
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

// Register puts the built-in vocabulary on r: the actions a spec can do, the
// assertions it can check, and the readers "read" and "waitFor" draw from.
// The grammar and interpreter (dsl) know none of these — a run wires
// them by calling this, and a test that wants a narrower vocabulary registers
// its own.
// Registry returns a fresh registry with the built-ins registered — the
// default vocabulary a run or `validate` resolves against.
func Registry() interp.Registry {
	r := interp.NewRegistry()
	Register(r)
	return r
}

func Register(r interp.Registry) {
	r.RegisterAction(actionSendTx, sendTxAction{})
	r.RegisterAction(actionWaitBlock, waitBlockAction{})
	r.RegisterAction(actionWaitFor, waitForAction{})
	r.RegisterAction(interp.ActionRead, readAction{})
	r.RegisterAction(actionNewAccount, newAccountAction{})
	r.RegisterAction(actionSendRawTampered, sendRawTamperedAction{})
	seedFaultBuiltins(r)
	seedAssetBuiltins(r)
	seedDerivedBuiltins(r)
	r.RegisterAssertion(assertBlockAdvance, blockAdvanceAssertion{})
	r.RegisterAssertion(assertSameBlockHash, sameBlockHashAssertion{})
	r.RegisterAssertion(assertMetric, metricAssertion{})
	r.RegisterAssertion(assertCallError, callErrorAssertion{})
	r.RegisterAssertion(assertMethodPresent, methodPresentAssertion{})
	for _, a := range builtinAssertions() {
		r.RegisterAssertion(a.name, a)
		r.RegisterReader(a.name, interp.Reader(a.read))
	}
	r.RegisterReader(assertTxMined, interp.Reader(readTxMined))
	r.RegisterAssertion(assertTxMined, txMinedAssertion{})
}

// waitBlockAction blocks until the target node's height reaches "target" (a
// number) or the timeout elapses. Args: target, on, timeout, pollInterval.
type waitBlockAction struct{}

func (waitBlockAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	target, ok := uintArg(ac.Args["target"])
	if !ok {
		return fmt.Errorf("dsl: waitBlock requires a numeric \"target\"")
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
			return fmt.Errorf("dsl: waitBlock: height %d not reached: %w", target, ctx.Err())
		case <-t.C:
		}
	}
}

// sendTxAction submits a node-signed transaction and, unless wait:false, polls
// for its receipt before returning.
type sendTxAction struct{}

// Do resolves the target node, sends the transaction, and waits for the
// receipt. Args: on (selector, default primary), from, to, value, gas, wait
// (default true), timeout, pollInterval. A negative expectation short-circuits
// the wait: expect:"reject" requires the submit itself to fail (see
// checkSubmitRejected), and expect:"revert"/expectRevert requires a mined
// status 0x0 (see checkTxOutcome).
func (sendTxAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	// A "key" arg switches to local signing: the tx is signed offline with that
	// private key and submitted via eth_sendRawTransaction, rather than
	// eth_sendTransaction against a node's keystore. This lets a spec send from a
	// non-node, non-governance account — the only way to exercise member-only
	// guards and blacklisted-sender paths, since every node coinbase is a member.
	if keyHex, ok := ac.Args["key"].(string); ok && keyHex != "" {
		return sendTxLocal(ctx, ac, keyHex)
	}
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	from, _ := ac.Args["from"].(string)
	if from == "" {
		return fmt.Errorf("dsl: sendTx requires \"from\"")
	}
	args := rpc.SendTxArgs{From: from}
	if to, ok := ac.Args["to"].(string); ok {
		args.To = to
	}
	if data, ok := ac.Args["data"].(string); ok {
		args.Data = data
	}
	// An access list (even []) selects a typed transaction: [] + gasPrice is an
	// EIP-2930 type 0x01. Passed through verbatim so the empty list survives.
	if al, ok := ac.Args["accessList"]; ok {
		args.AccessList = al
	}
	if v, ok := hexQuantity(ac.Args["value"]); ok {
		args.Value = v
	}
	if g, ok := hexQuantity(ac.Args["gas"]); ok {
		args.Gas = g
	}
	if err := applyFeeArgs(&args, ac.Args); err != nil {
		return err
	}
	hash, err := c.SendTransaction(ctx, args)
	// An expect:"reject" step inverts the submit outcome: the node must refuse
	// the transaction at submit time (no hash). It is checked before the receipt
	// wait because a rejected tx never enters a block.
	if wantReject(ac.Args) {
		return checkSubmitRejected(hash, err, ac)
	}
	if err != nil {
		return fmt.Errorf("dsl: sendTx: %w", err)
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

// sendTxLocal signs and submits a transaction locally with the given private
// key (hex, optional 0x prefix), routing the outcome through the same
// reject/wait/revert logic as the node-signed path. It uses the injected
// account provider's Wallet: a "feePayerKey" arg makes it a 0x16 fee-delegated
// transfer (the "key" account is the sender, feePayerKey covers the gas),
// Execute when a "data" payload is present, else SendCoin for a value-only
// transfer. The target node RPC comes from the usual "on" selector so the wallet
// dials the same endpoint the node-signed path would.
func sendTxLocal(ctx context.Context, ac *interp.ActionCtx, keyHex string) error {
	if ac.Deps == nil || ac.Deps.Accounts == nil {
		return fmt.Errorf("dsl: sendTx key: no account provider")
	}
	priv, err := hex.DecodeString(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return fmt.Errorf("dsl: sendTx key: decode: %w", err)
	}
	rpcURL := selectorTarget(ac.Env, ac.Args)
	w, err := ac.Deps.Accounts.OpenWallet(ctx, priv, rpcURL)
	if err != nil {
		return fmt.Errorf("dsl: sendTx: open wallet: %w", err)
	}
	to, _ := ac.Args["to"].(string)
	if to == "" {
		return fmt.Errorf("dsl: sendTx key requires \"to\"")
	}
	value, err := parseValueWei(ac.Args["value"])
	if err != nil {
		return err
	}
	feePayerKey, _ := ac.Args["feePayerKey"].(string)
	data, _ := ac.Args["data"].(string)
	var hash string
	switch {
	// A "feePayerKey" arg makes this a 0x16 fee-delegated transfer: the "key"
	// account signs as the sender (moves value) while feePayerKey covers the gas.
	// It is the only way to exercise a blacklisted-fee-payer rejection — the SDK's
	// static value-transfer guard checks sender and recipient but not the fee
	// payer, so the tx reaches the node, which is what does the rejecting.
	case feePayerKey != "":
		fp, ferr := hex.DecodeString(strings.TrimPrefix(feePayerKey, "0x"))
		if ferr != nil {
			return fmt.Errorf("dsl: sendTx feePayerKey: decode: %w", ferr)
		}
		if value == nil {
			value = new(big.Int)
		}
		hash, err = w.SendFeeDelegated(ctx, fp, to, value)
	case data != "" && data != "0x":
		b, derr := hex.DecodeString(strings.TrimPrefix(data, "0x"))
		if derr != nil {
			return fmt.Errorf("dsl: sendTx: data: %w", derr)
		}
		hash, err = w.Execute(ctx, to, b, value)
	default:
		if value == nil {
			value = new(big.Int)
		}
		// Explicit fee caps ride a type-0x02 send; without them the wallet
		// suggests fees (which on an anzeon chain means the forced gasTip, so
		// a case probing the authorized-account exemption must set its own).
		// An explicit nonce (out-of-order / replacement cases) forces the same
		// fully-specified send, since SendCoin auto-picks the nonce.
		maxFee, hasMaxFee := hexQuantity(ac.Args["maxFeePerGas"])
		tip, hasTip := hexQuantity(ac.Args["maxPriorityFeePerGas"])
		if hasMaxFee != hasTip {
			return fmt.Errorf("dsl: sendTx key: maxFeePerGas and maxPriorityFeePerGas come together")
		}
		noncePtr, nerr := noncePointer(ac.Args["nonce"])
		if nerr != nil {
			return nerr
		}
		if hasMaxFee || noncePtr != nil {
			dargs := accounts.DynamicTxArgs{ToHex: to, Value: value, Gas: 21000, Nonce: noncePtr}
			if hasMaxFee {
				feeCap, ok1 := new(big.Int).SetString(strings.TrimPrefix(maxFee, "0x"), 16)
				tipCap, ok2 := new(big.Int).SetString(strings.TrimPrefix(tip, "0x"), 16)
				if !ok1 || !ok2 {
					return fmt.Errorf("dsl: sendTx key: bad fee quantity")
				}
				dargs.GasFeeCap, dargs.GasTipCap = feeCap, tipCap
			}
			hash, err = w.SendDynamicFeeTx(ctx, dargs)
			break
		}
		hash, err = w.SendCoin(ctx, to, value)
	}
	if wantReject(ac.Args) {
		return checkSubmitRejected(hash, err, ac)
	}
	if err != nil {
		return fmt.Errorf("dsl: sendTx: %w", err)
	}
	ac.Hash = hash
	if wait, ok := ac.Args["wait"].(bool); ok && !wait {
		return nil
	}
	c, err := clientFor(ac.Deps, rpcURL)
	if err != nil {
		return err
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

// noncePointer reads an optional "nonce" arg (decimal or 0x-hex) as a *uint64,
// returning nil when absent so the wallet auto-picks the next nonce. An explicit
// nonce is what lets a spec submit out of order or replace a pending tx.
func noncePointer(v any) (*uint64, error) {
	q, ok := hexQuantity(v)
	if !ok {
		return nil, nil
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(q, "0x"), 16)
	if !ok || n.Sign() < 0 || !n.IsUint64() {
		return nil, fmt.Errorf("dsl: sendTx: bad nonce %v", v)
	}
	u := n.Uint64()
	return &u, nil
}

// parseValueWei reads a tx "value" arg (decimal or 0x-hex wei) as a big.Int,
// returning nil when absent. It reuses hexQuantity so the accepted forms match
// the node-signed path exactly.
func parseValueWei(v any) (*big.Int, error) {
	q, ok := hexQuantity(v)
	if !ok {
		return nil, nil
	}
	bi, ok := new(big.Int).SetString(strings.TrimPrefix(q, "0x"), 16)
	if !ok {
		return nil, fmt.Errorf("dsl: sendTx: bad value %v", v)
	}
	return bi, nil
}

// newAccountAction generates a fresh key pair off-chain and binds it for later
// steps: the address under "save" (referenceable as "$name") and the private
// key hex under "saveKey". The key is an ephemeral, throwaway test key — a spec
// funds it from a node account, then uses it as sendTx "key" to sign locally.
type newAccountAction struct{}

// Do generates a key pair and records the address as the step value plus the
// private key under the "saveKey" binding. Args: save (address binding, the
// usual step "save"), saveKey (private-key binding name, required).
func (newAccountAction) Do(_ context.Context, ac *interp.ActionCtx) error {
	keyName, _ := ac.Args["saveKey"].(string)
	if keyName == "" {
		return fmt.Errorf("dsl: newAccount requires \"saveKey\"")
	}
	priv, addr, err := accounts.GenerateKey()
	if err != nil {
		return fmt.Errorf("dsl: newAccount: %w", err)
	}
	ac.Value = addr
	if ac.Extra == nil {
		ac.Extra = map[string]any{}
	}
	ac.Extra[keyName] = hex.EncodeToString(priv)
	return nil
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
			return fmt.Errorf("dsl: sendTx %s expected revert but succeeded", hash)
		}
		return nil
	}
	if reverted {
		return fmt.Errorf("dsl: sendTx %s reverted (status 0x0)", hash)
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

// wantReject reports whether a tx step declares that it expects a submit-time
// rejection, via expectReject:true or expect:"reject". A rejection (the node
// refuses the transaction before it is mined) is distinct from a revert (the
// transaction is mined with status 0x0, handled by wantRevert).
func wantReject(args map[string]any) bool {
	if b, ok := args["expectReject"].(bool); ok {
		return b
	}
	if s, ok := args["expect"].(string); ok {
		return strings.EqualFold(s, "reject")
	}
	return false
}

// checkSubmitRejected enforces an expect:"reject" step: the node must refuse the
// transaction at submit time (SendTransaction returns an error and no hash). An
// accepted submit fails the step. An optional "reason" (case-insensitive
// substring) tightens the check to a specific rejection message, so a spec can
// require e.g. an "insufficient funds" rejection rather than any error.
func checkSubmitRejected(hash string, submitErr error, ac *interp.ActionCtx) error {
	if submitErr == nil {
		return fmt.Errorf("dsl: sendTx expected submit rejection but the node accepted it (hash %s)", hash)
	}
	if reason, ok := ac.Args["reason"].(string); ok && reason != "" {
		if !strings.Contains(strings.ToLower(submitErr.Error()), strings.ToLower(reason)) {
			return fmt.Errorf("dsl: sendTx was rejected but not for %q: %v", reason, submitErr)
		}
	}
	// Bind the rejection message so a "save" can surface it to a later assertion.
	ac.Value = submitErr.Error()
	return nil
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
				return nil, fmt.Errorf("dsl: sendTx: parse receipt %s: %w", hash, uerr)
			}
			return m, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("dsl: sendTx: receipt %s: %w", hash, ctx.Err())
		case <-t.C:
		}
	}
}

// clientFor returns an RPC client for url, guarding a missing injected factory.
func clientFor(deps *interp.Deps, url string) (*rpc.Client, error) {
	if deps == nil || deps.RPC == nil {
		return nil, fmt.Errorf("dsl: no RPC client injected")
	}
	if url == "" {
		return nil, fmt.Errorf("dsl: no target node RPC URL")
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
func assertTargets(ac *interp.AssertCtx) []assertTarget {
	if len(ac.On) > 0 {
		out := make([]assertTarget, 0, len(ac.On))
		for _, n := range ac.On {
			out = append(out, assertTarget{name: string(node.LabelFor(n.Index)), url: n.RPCURL})
		}
		return out
	}
	if ac.Env != nil {
		if nodes := ac.Env.Nodes(); len(nodes) > 0 {
			return []assertTarget{{name: string(node.LabelFor(nodes[0].Index)), url: nodes[0].RPCURL}}
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
