package testhelper

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/assert"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// This file holds the tx-probe vocabulary the nonce, fee-delegation-tamper,
// eth_call-revert, and RPC-presence cases needed — the primitives the legacy
// retirement plan (R5) listed as gaps. None of it names a chain: the tamper
// action asks the injected account provider whether it supports 0x16, and the
// method probe takes the method name from the spec.

// readTxMined returns "true" when a transaction has a receipt (is on chain) and
// "false" otherwise. It is a plain read, not a wait, so a spec can both wait for
// one tx to mine (waitFor source:txMined compare:Equal expected:"true") and
// assert another did not (assert txMined expected:"false") — the replacement-tx
// scenario, where the replaced transaction must never mine.
func readTxMined(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	hash, ok := spec["hash"].(string)
	if !ok || hash == "" {
		return nil, fmt.Errorf("dsl: txMined requires \"hash\"")
	}
	raw, err := c.TxReceipt(ctx, hash)
	if err != nil {
		return nil, err
	}
	if raw == nil || strings.TrimSpace(string(raw)) == "null" {
		return "false", nil
	}
	return "true", nil
}

// txMinedAssertion checks whether a transaction is on chain against an expected
// "true"/"false". It reads once, so pair it with a waitFor when the positive
// outcome needs settling time; the negative outcome (the replaced tx) is stable
// once its competitor has mined.
type txMinedAssertion struct{}

func (txMinedAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertTxMined, Provenance: ac.Spec, Pass: true}
	expected := ac.Spec["expected"]
	res.Expected = expected
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: txMined: no target node RPC URL")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	c, err := clientFor(ac.Deps, targets[0].url)
	if err != nil {
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	actual, err := readTxMined(ctx, c, ac.Spec)
	if err != nil {
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	res.Actual = actual
	if pass, _ := assert.Equal(actual, expected); !pass {
		res.Pass = false
	}
	return res, nil
}

// sendRawTamperedAction builds a 0x16 fee-delegated transfer, corrupts one of
// its two signatures, submits the raw bytes, and requires the node to reject
// them. Args: senderKey, feePayerKey (hex, 0x optional), to, value (default 1),
// which ("sender" | "feepayer"), on (target selector). The chain id and the
// sender's nonce come from the node; the fee caps are read from the head so the
// only reason the node can refuse the tx is the broken signature.
type sendRawTamperedAction struct{}

func (sendRawTamperedAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	if ac.Deps == nil || ac.Deps.Accounts == nil {
		return fmt.Errorf("dsl: sendRawTampered: no account provider")
	}
	if !ac.Deps.Accounts.SupportsTxType(0x16) {
		return fmt.Errorf("dsl: sendRawTampered: chain does not support fee delegation (0x16)")
	}
	which, _ := ac.Args["which"].(string)
	if which != "sender" && which != "feepayer" {
		return fmt.Errorf("dsl: sendRawTampered requires \"which\" of \"sender\" or \"feepayer\"")
	}
	senderKey, err := hexKey(ac.Args["senderKey"], "senderKey")
	if err != nil {
		return err
	}
	feePayerKey, err := hexKey(ac.Args["feePayerKey"], "feePayerKey")
	if err != nil {
		return err
	}
	to, _ := ac.Args["to"].(string)
	if to == "" {
		return fmt.Errorf("dsl: sendRawTampered requires \"to\"")
	}
	value, err := parseValueWei(ac.Args["value"])
	if err != nil {
		return err
	}
	if value == nil {
		value = big.NewInt(1)
	}

	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	chainID, err := c.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("dsl: sendRawTampered: chain id: %w", err)
	}
	senderAddr, err := accounts.AddressOf(senderKey)
	if err != nil {
		return fmt.Errorf("dsl: sendRawTampered: %w", err)
	}
	nonce, err := c.NonceAt(ctx, senderAddr)
	if err != nil {
		return fmt.Errorf("dsl: sendRawTampered: nonce: %w", err)
	}
	feeCap, tipCap, err := validFeesFor(ctx, c)
	if err != nil {
		return err
	}

	raw, err := accounts.EncodeFeeDelegatedTampered(senderKey, feePayerKey, to, value,
		int64(chainID), nonce, feeCap, tipCap, which)
	if err != nil {
		return fmt.Errorf("dsl: sendRawTampered: build: %w", err)
	}
	var h string
	sendErr := c.Call(ctx, "eth_sendRawTransaction", &h, "0x"+hex.EncodeToString(raw))
	if sendErr == nil {
		return fmt.Errorf("dsl: sendRawTampered: node accepted a tx with a corrupt %s signature (hash %s)", which, h)
	}
	ac.Value = sendErr.Error()
	return nil
}

// sendSetCodeAction sends an EIP-7702 (type 0x04) set-code transaction: the
// sponsor (key) pays gas to delegate a fresh authority's account code to a
// fixed address. Args: key (sponsor, hex), authorityKey (the delegating
// account, hex), delegate (the address the authority's code points at), on. The
// tx hash is bound under "save"; a spec then asserts the tx type is 0x4 and the
// authority's code became the 0xef0100||delegate indicator.
type sendSetCodeAction struct{}

func (sendSetCodeAction) Do(ctx context.Context, ac *interp.ActionCtx) error {
	if ac.Deps == nil || ac.Deps.Accounts == nil {
		return fmt.Errorf("dsl: sendSetCode: no account provider")
	}
	if !ac.Deps.Accounts.SupportsTxType(0x04) {
		return fmt.Errorf("dsl: sendSetCode: chain does not support set-code (0x04)")
	}
	sponsorKey, err := hexKeyArg(ac.Args["key"], "key")
	if err != nil {
		return err
	}
	authorityKey, err := hexKeyArg(ac.Args["authorityKey"], "authorityKey")
	if err != nil {
		return err
	}
	delegate, _ := ac.Args["delegate"].(string)
	if delegate == "" {
		return fmt.Errorf("dsl: sendSetCode requires \"delegate\"")
	}
	rpcURL := selectorTarget(ac.Env, ac.Args)
	w, err := ac.Deps.Accounts.OpenWallet(ctx, sponsorKey, rpcURL)
	if err != nil {
		return fmt.Errorf("dsl: sendSetCode: open sponsor wallet: %w", err)
	}
	hash, err := w.SendSetCode(ctx, authorityKey, delegate)
	if err != nil {
		return fmt.Errorf("dsl: sendSetCode: %w", err)
	}
	ac.Value = hash
	ac.Hash = hash
	return nil
}

// hexKeyArg decodes a spec key argument (hex, 0x optional) into raw bytes, for
// the set-code action's sponsor and authority keys.
func hexKeyArg(v any, name string) ([]byte, error) {
	s, _ := v.(string)
	if s == "" {
		return nil, fmt.Errorf("dsl: sendSetCode requires %q", name)
	}
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, fmt.Errorf("dsl: sendSetCode: %s: %w", name, err)
	}
	return b, nil
}

// hexKey decodes a spec key argument (hex, 0x optional) into raw bytes.
func hexKey(v any, name string) ([]byte, error) {
	s, _ := v.(string)
	if s == "" {
		return nil, fmt.Errorf("dsl: sendRawTampered requires %q", name)
	}
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, fmt.Errorf("dsl: sendRawTampered: %s: %w", name, err)
	}
	return b, nil
}

// validFeesFor reads a comfortably-valid (feeCap, tipCap) from the head: feeCap
// is 2*baseFee + tip, so an ordinary send would be accepted and the tamper is
// the only reason to reject.
func validFeesFor(ctx context.Context, c *rpc.Client) (feeCap, tipCap *big.Int, err error) {
	b, err := c.BlockByNumber(ctx, "latest")
	if err != nil {
		return nil, nil, fmt.Errorf("dsl: sendRawTampered: base fee: %w", err)
	}
	base := b.BaseFeePerGas
	if base == nil {
		base = new(big.Int)
	}
	tip := big.NewInt(1_000_000_000) // 1 gwei, a comfortable default
	var tipHex string
	if err := c.Call(ctx, "eth_maxPriorityFeePerGas", &tipHex); err == nil {
		if v, ok := new(big.Int).SetString(strings.TrimPrefix(tipHex, "0x"), 16); ok && v.Sign() > 0 {
			tip = v
		}
	}
	return new(big.Int).Add(new(big.Int).Mul(base, big.NewInt(2)), tip), tip, nil
}

// callErrorAssertion passes when an eth_call returns an error — the eth_call
// revert path, where a call to a reverting function must surface a JSON-RPC
// error rather than a value. Args: to, data, on. A successful call fails the
// assertion.
type callErrorAssertion struct{}

func (callErrorAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertCallError, Provenance: ac.Spec, Pass: true}
	res.Expected = "eth_call returns an error"
	to, _ := ac.Spec["to"].(string)
	data, _ := ac.Spec["data"].(string)
	if to == "" || data == "" {
		err := fmt.Errorf("dsl: callError requires \"to\" and \"data\"")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: callError: no target node RPC URL")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	c, err := clientFor(ac.Deps, targets[0].url)
	if err != nil {
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	out, callErr := c.EthCall(ctx, to, data)
	if callErr == nil {
		res.Pass, res.Actual = false, out
		res.Source = "eth_call returned a value, want an error"
		return res, nil
	}
	res.Actual = callErr.Error()
	return res, nil
}

// methodPresentAssertion passes when a JSON-RPC method is registered on the
// node — the call may error for a throwaway argument, but a method-not-found
// (-32601) means the method is absent. Args: method (required), params
// (optional array), on. It is the indirect presence check for a
// chain-distinctive RPC whose full flow another case exercises.
type methodPresentAssertion struct{}

func (methodPresentAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	res := session.AssertResult{Assert: assertMethodPresent, Provenance: ac.Spec, Pass: true}
	method, _ := ac.Spec["method"].(string)
	if method == "" {
		err := fmt.Errorf("dsl: methodPresent requires \"method\"")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	res.Expected = method + " is a registered method"
	var params []any
	if raw, ok := ac.Spec["params"].([]any); ok {
		params = raw
	}
	targets := assertTargets(ac)
	if len(targets) == 0 {
		err := fmt.Errorf("dsl: methodPresent: no target node RPC URL")
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	c, err := clientFor(ac.Deps, targets[0].url)
	if err != nil {
		res.Pass, res.Actual = false, err.Error()
		return res, err
	}
	var out any
	callErr := c.Call(ctx, method, &out, params...)
	if isMethodNotFound(callErr) {
		res.Pass, res.Actual = false, callErr.Error()
		res.Source = "method not found"
		return res, nil
	}
	if callErr != nil {
		res.Actual = "present (rejected the probe argument: " + callErr.Error() + ")"
	} else {
		res.Actual = "present"
	}
	return res, nil
}

// isMethodNotFound reports whether err is a JSON-RPC "method not found" response
// (code -32601 or a client's phrasing of it), as opposed to any other error the
// method might legitimately return for a throwaway argument.
func isMethodNotFound(err error) bool {
	if err == nil {
		return false
	}
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "-32601") ||
		strings.Contains(m, "method not found") ||
		strings.Contains(m, "does not exist") ||
		strings.Contains(m, "not available")
}
