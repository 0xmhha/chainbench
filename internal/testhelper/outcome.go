package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// What a submitted transaction did, and the argument shapes a step declares it
// with.
//
// A spec can expect a receipt, a revert, or a refusal to admit the transaction
// at all, and the three are different events: a revert is a mined transaction
// that failed, a rejection never reached a block. Telling them apart is the
// whole value of asking.

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

// defaultKeptOutBlocks is how far a keptOut step watches before it believes the
// transaction is staying out.
//
// A block is produced every second on all three chains, so five is five seconds
// of a producer having every chance to take it. The number is small on purpose:
// this outcome is only declared where the chain cannot include the transaction
// at all — a fee cap under the base fee is refused by the state transition
// itself — so waiting longer would only make a passing run slower.
const defaultKeptOutBlocks = 5

// checkKeptOut enforces an expect:"keptOut" step: the transaction must not reach
// a block. It is satisfied two ways, and BOTH are checked rather than either
// being a free pass.
//
// A pool that refuses the transaction at submit satisfies it immediately, and
// the optional "reason" still has to match — a refusal for some other cause is
// not this outcome. A pool that accepts it satisfies it only by never mining
// it, which is watched for "blocks" blocks and fails the moment a receipt
// appears.
//
// The two shapes are one rule seen from two pools. A transaction priced under
// the base fee cannot be executed on any of the three chains; whether its pool
// says so at submit or lets it sit is the chain's own manner, and a case that
// insisted on one manner would be asserting the manner rather than the rule.
func checkKeptOut(ctx context.Context, c *rpc.Client, hash string, submitErr error, ac *interp.ActionCtx) error {
	if submitErr != nil {
		// Refused at submit: the same reason check a reject step gets.
		return checkSubmitRejected(hash, submitErr, ac)
	}
	ac.Hash = hash
	blocks := uint64(defaultKeptOutBlocks)
	if n, ok := uintArg(ac.Args["blocks"]); ok && n > 0 {
		blocks = n
	}
	start, err := c.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("dsl: sendTx keptOut: read head: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, durationArg(ac.Args, "timeout", defaultTxTimeout))
	defer cancel()
	t := time.NewTicker(durationArg(ac.Args, "pollInterval", defaultTxPollInterval))
	defer t.Stop()
	for {
		raw, rerr := c.TxReceipt(ctx, hash)
		if rerr == nil && raw != nil && strings.TrimSpace(string(raw)) != "null" {
			return fmt.Errorf("dsl: sendTx %s was accepted and mined, but this chain must keep it out of a block", hash)
		}
		head, herr := c.BlockNumber(ctx)
		if herr == nil && head >= start+blocks {
			ac.Value = "kept out for " + strconv.FormatUint(blocks, 10) + " block(s)"
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("dsl: sendTx keptOut: %d block(s) did not pass within the timeout: %w", blocks, ctx.Err())
		case <-t.C:
		}
	}
}

// wantKeptOut reports whether a tx step declares the keptOut outcome.
func wantKeptOut(args map[string]any) bool {
	s, ok := args["expect"].(string)
	return ok && strings.EqualFold(s, "keptOut")
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
func selectorTarget(env interp.NodeTable, args map[string]any) string {
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
		// Decimal or 0x-hex. Every block number, gas figure and nonce the chain
		// hands back is 0x-hex, so decimal-only made a value read from the chain
		// unusable in the very args that take one — waitBlock could not be given
		// a receipt's block number. parseBigValue has always taken both; this is
		// the same rule, not a new one.
		t := strings.TrimSpace(x)
		if h, ok := strings.CutPrefix(t, "0x"); ok {
			n, err := strconv.ParseUint(h, 16, 64)
			return n, err == nil
		}
		n, err := strconv.ParseUint(t, 10, 64)
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
