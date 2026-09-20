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
