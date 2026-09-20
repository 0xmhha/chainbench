package testhelper

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/internal/dsl/interp"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// Sending a transaction, by the node's key or by a local one.
//
// The local path is the one a public endpoint needs: a real network's node does
// not unlock somebody else's account, so a test that must run there signs for
// itself. Both paths end at the same receipt wait, which keeps a spec from
// having to know which one it took.

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
	fromRef, _ := ac.Args["from"].(string)
	if fromRef == "" {
		return fmt.Errorf("dsl: sendTx requires \"from\"")
	}
	sender, err := ResolveAccount(ac.Deps, fromRef)
	if err != nil {
		return err
	}
	// An account only this harness holds cannot be signed by a node: the node
	// has no such key, and asking it to would fail with "unknown account",
	// which says nothing about why. Signing here is not a fallback — it is the
	// only place that key exists.
	if sender.SignsLocally() {
		return sendTxLocalFrom(ctx, ac, sender)
	}
	args := rpc.SendTxArgs{From: sender.Address}
	if to, ok := ac.Args["to"].(string); ok {
		// A recipient may be a contract, which has no key. Only "from" has to be
		// an account here.
		addr, rerr := ResolveAddress(ac.Deps, to)
		if rerr != nil {
			return rerr
		}
		args.To = addr
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
// sendTxLocalFrom signs with the key a label resolved to. It is the same path
// an explicit "key" takes; the difference is only where the key came from.
func sendTxLocalFrom(ctx context.Context, ac *interp.ActionCtx, from Account) error {
	return sendTxLocalKey(ctx, ac, from.Key)
}

func sendTxLocal(ctx context.Context, ac *interp.ActionCtx, keyHex string) error {
	priv, err := hex.DecodeString(strings.TrimPrefix(keyHex, "0x"))
	if err != nil {
		return fmt.Errorf("dsl: sendTx key: decode: %w", err)
	}
	return sendTxLocalKey(ctx, ac, priv)
}

func sendTxLocalKey(ctx context.Context, ac *interp.ActionCtx, priv []byte) error {
	if ac.Deps == nil || ac.Deps.Accounts == nil {
		return fmt.Errorf("dsl: sendTx: no account provider")
	}
	rpcURL := selectorTarget(ac.Env, ac.Args)
	w, err := ac.Deps.Accounts.OpenWallet(ctx, priv, rpcURL)
	if err != nil {
		return fmt.Errorf("dsl: sendTx: open wallet: %w", err)
	}
	toRef, _ := ac.Args["to"].(string)
	if toRef == "" {
		return fmt.Errorf("dsl: sendTx requires \"to\" when this harness signs")
	}
	to, err := ResolveAddress(ac.Deps, toRef)
	if err != nil {
		return err
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
