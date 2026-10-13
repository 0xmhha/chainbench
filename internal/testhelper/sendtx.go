package testhelper

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
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
	if _, supplied := ac.Args["feePayer"]; supplied {
		return fmt.Errorf("dsl: feePayer requires a local signing account")
	}
	if _, supplied := ac.Args["feePayerKey"]; supplied {
		return fmt.Errorf("dsl: feePayerKey requires a local signing account")
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
	if wantKeptOut(ac.Args) {
		return checkKeptOut(ctx, c, hash, err, ac)
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
	receipt, err := waitReceipt(ctx, c, hash,
		durationArg(ac.Args, "timeout", defaultTxTimeout),
		durationArg(ac.Args, "pollInterval", defaultTxPollInterval))
	if err != nil {
		return err
	}
	ac.Receipt = receipt
	return checkTxOutcome(hash, receipt, ac.Args)
}

// sendTxLocalFrom signs and submits a transaction locally with the given private
// key (hex, optional 0x prefix), routing the outcome through the same
// reject/wait/revert logic as the node-signed path. It uses the injected
// account provider's Wallet: a "feePayerKey" arg makes it a 0x16 fee-delegated
// transfer (the "key" account is the sender, feePayerKey covers the gas), a
// "gasPrice" makes it a fully specified legacy transaction (or, with an
// accessList, an EIP-2930 transaction preserving the list's entries),
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
	if err := checkLocalSendArgs(ac.Args); err != nil {
		return err
	}
	feePayerKey, _ := ac.Args["feePayerKey"].(string)
	feePayer, _ := ac.Args["feePayer"].(string)
	data, _ := ac.Args["data"].(string)
	gasHex, hasGas := hexQuantity(ac.Args["gas"])
	_, hasGasPrice := hexQuantity(ac.Args["gasPrice"])
	var hash string
	switch {
	// A "gasPrice" names the pre-1559 fee field, so the case asked for a type
	// that carries it: legacy, or EIP-2930 when an access list comes with it.
	case hasGasPrice:
		hash, err = sendLegacyOptions(ctx, ac, w, to, value, data, rpcURL)
	// A "feePayerKey" arg makes this a 0x16 fee-delegated transfer: the "key"
	// account signs as the sender (moves value) while feePayerKey covers the gas.
	// It is the only way to exercise a blacklisted-fee-payer rejection — the SDK's
	// static value-transfer guard checks sender and recipient but not the fee
	// payer, so the tx reaches the node, which is what does the rejecting.
	case feePayerKey != "" || feePayer != "":
		hash, err = sendDelegatedOptions(ctx, ac, w, to, value, data, rpcURL)
	case ac.Args["accessList"] != nil:
		hash, err = sendDynamicAccessList(ctx, ac, w, to, value, data, rpcURL)
	// An explicit "gas" is the limit the case asked for. Every other branch
	// picks its own — SendCoin fixes 21000, Execute estimates — so a case that
	// sent 100000 gas to a reverting contract ran out at 21000 instead, and
	// measured out-of-gas where it meant to measure a revert.
	case hasGas:
		hash, err = sendWithGas(ctx, ac, w, to, value, data, gasHex, rpcURL)
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
	if wantKeptOut(ac.Args) {
		// This path signs locally and has no client of its own; keptOut has to
		// watch the same node the wallet posted to.
		c, cerr := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
		if cerr != nil {
			return cerr
		}
		return checkKeptOut(ctx, c, hash, err, ac)
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

// legacyExcludes rejects mixing the two fee models in one transaction.
var legacyExcludes = []string{"maxFeePerGas", "maxPriorityFeePerGas"}

// checkLocalSendArgs refuses a combination the local signing path would
// otherwise send without one of its fields. A case that names a field expects
// the transaction to carry it, so dropping it silently measures something else.
func checkLocalSendArgs(args map[string]any) error {
	_, hasPayer := args["feePayer"]
	_, hasPayerKey := args["feePayerKey"]
	if hasPayer && hasPayerKey {
		return fmt.Errorf("dsl: feePayer and feePayerKey are mutually exclusive")
	}
	if hasPayer {
		if name, ok := args["feePayer"].(string); !ok || strings.TrimSpace(name) == "" {
			return fmt.Errorf("dsl: feePayer must name a local account")
		}
	}
	if hasPayerKey {
		if key, ok := args["feePayerKey"].(string); !ok || key == "" {
			return fmt.Errorf("dsl: feePayerKey must name a private key binding")
		}
	}
	if hasPayer || hasPayerKey {
		if _, legacy := args["gasPrice"]; legacy {
			return fmt.Errorf("dsl: fee delegation cannot carry gasPrice")
		}
	}
	al, hasList := args["accessList"]
	_, hasGasPrice := args["gasPrice"]
	if hasList {
		if _, ok := al.([]any); !ok {
			return fmt.Errorf("dsl: sendTx key: accessList must be an array")
		}
	}
	if !hasGasPrice {
		return nil
	}
	for _, f := range legacyExcludes {
		if _, has := args[f]; has {
			return fmt.Errorf("dsl: sendTx key: gasPrice cannot be combined with %q", f)
		}
	}
	return nil
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

// sendWithGas retains an explicit legacy gas price, or signs type 0x02 with
// the named gas limit, data, and fee caps — else the node's suggestion:
// tip from eth_maxPriorityFeePerGas, cap twice the latest base fee plus the tip.
func sendWithGas(ctx context.Context, ac *interp.ActionCtx, w accounts.Wallet, to string, value *big.Int, data, gasHex, rpcURL string) (string, error) {
	return sendWithAccessListGas(ctx, ac, w, to, value, data, gasHex, rpcURL, nil)
}

func sendWithAccessListGas(ctx context.Context, ac *interp.ActionCtx, w accounts.Wallet, to string, value *big.Int, data, gasHex, rpcURL string, list []accounts.AccessTuple) (string, error) {
	return sendSpecifiedGas(ctx, ac, w, to, value, data, gasHex, rpcURL, list, nil)
}

func sendSpecifiedGas(ctx context.Context, ac *interp.ActionCtx, w accounts.Wallet, to string, value *big.Int, data, gasHex, rpcURL string, list []accounts.AccessTuple, feeKey []byte) (string, error) {
	gas, err := strconv.ParseUint(strings.TrimPrefix(gasHex, "0x"), 16, 64)
	if err != nil {
		return "", fmt.Errorf("dsl: sendTx gas: %w", err)
	}
	if value == nil {
		value = new(big.Int)
	}
	args := accounts.DynamicTxArgs{ToHex: to, Value: value, Gas: gas, AccessList: list}
	if data != "" && data != "0x" {
		b, derr := hex.DecodeString(strings.TrimPrefix(data, "0x"))
		if derr != nil {
			return "", fmt.Errorf("dsl: sendTx: data: %w", derr)
		}
		args.Data = b
	}
	nonce, err := noncePointer(ac.Args["nonce"])
	if err != nil {
		return "", err
	}
	args.Nonce = nonce
	if price, legacy := hexQuantity(ac.Args["gasPrice"]); legacy {
		if _, supplied := ac.Args["maxFeePerGas"]; supplied {
			return "", fmt.Errorf("dsl: gasPrice and maxFeePerGas are mutually exclusive")
		}
		if _, supplied := ac.Args["maxPriorityFeePerGas"]; supplied {
			return "", fmt.Errorf("dsl: gasPrice and maxPriorityFeePerGas are mutually exclusive")
		}
		gasPrice, ok := new(big.Int).SetString(strings.TrimPrefix(price, "0x"), 16)
		if !ok {
			return "", fmt.Errorf("dsl: invalid gasPrice")
		}
		return w.SendLegacyTx(ctx, accounts.LegacyTxArgs{ToHex: to, Value: value, Data: args.Data, Gas: gas, GasPrice: gasPrice, Nonce: nonce, AccessList: list})
	}
	maxFee, hasMaxFee := hexQuantity(ac.Args["maxFeePerGas"])
	tip, hasTip := hexQuantity(ac.Args["maxPriorityFeePerGas"])
	if hasMaxFee != hasTip {
		return "", fmt.Errorf("dsl: sendTx key: maxFeePerGas and maxPriorityFeePerGas come together")
	}
	if hasMaxFee {
		feeCap, ok1 := new(big.Int).SetString(strings.TrimPrefix(maxFee, "0x"), 16)
		tipCap, ok2 := new(big.Int).SetString(strings.TrimPrefix(tip, "0x"), 16)
		if !ok1 || !ok2 {
			return "", fmt.Errorf("dsl: sendTx key: bad fee quantity")
		}
		args.GasFeeCap, args.GasTipCap = feeCap, tipCap
	} else {
		c, cerr := clientFor(ac.Deps, rpcURL)
		if cerr != nil {
			return "", cerr
		}
		var tipHex string
		if err := c.Call(ctx, "eth_maxPriorityFeePerGas", &tipHex); err != nil {
			return "", fmt.Errorf("dsl: sendTx: suggested tip: %w", err)
		}
		tipCap, ok := new(big.Int).SetString(strings.TrimPrefix(tipHex, "0x"), 16)
		if !ok {
			return "", fmt.Errorf("dsl: sendTx: bad suggested tip %q", tipHex)
		}
		blk, berr := c.BlockByNumber(ctx, "latest")
		if berr != nil {
			return "", fmt.Errorf("dsl: sendTx: base fee: %w", berr)
		}
		base := blk.BaseFeePerGas
		if base == nil {
			base = new(big.Int)
		}
		args.GasTipCap = tipCap
		args.GasFeeCap = new(big.Int).Add(new(big.Int).Mul(base, big.NewInt(2)), tipCap)
	}
	if feeKey != nil {
		return w.SendFeeDelegatedTx(ctx, feeKey, args)
	}
	return w.SendDynamicFeeTx(ctx, args)
}

// sendLegacyOptions is exclusive to sendTx: asset actions keep their own schema.
func sendLegacyOptions(ctx context.Context, ac *interp.ActionCtx, w accounts.Wallet, to string, value *big.Int, data, rpcURL string) (string, error) {
	var list []accounts.AccessTuple
	if raw, supplied := ac.Args["accessList"]; supplied {
		decoded, err := localAccessList(ac.Deps, raw)
		if err != nil {
			return "", err
		}
		list = decoded
	}
	gas := uint64(21000)
	if quantity, explicit := hexQuantity(ac.Args["gas"]); explicit {
		parsed, err := strconv.ParseUint(strings.TrimPrefix(quantity, "0x"), 16, 64)
		if err != nil {
			return "", fmt.Errorf("dsl: gas: %w", err)
		}
		gas = parsed
	} else if (data != "" && data != "0x") || len(list) > 0 {
		client, err := clientFor(ac.Deps, rpcURL)
		if err != nil {
			return "", err
		}
		message := map[string]any{"from": w.Address(), "to": to, "data": data}
		if value != nil {
			message["value"] = "0x" + value.Text(16)
		}
		if list != nil {
			message["accessList"] = list
		}
		var quantity string
		if err := client.Call(ctx, "eth_estimateGas", &quantity, message); err != nil {
			return "", err
		}
		gas, err = strconv.ParseUint(strings.TrimPrefix(quantity, "0x"), 16, 64)
		if err != nil {
			return "", err
		}
	}
	price, _ := hexQuantity(ac.Args["gasPrice"])
	gasPrice, ok := new(big.Int).SetString(strings.TrimPrefix(price, "0x"), 16)
	if !ok {
		return "", fmt.Errorf("dsl: invalid gasPrice")
	}
	nonce, err := noncePointer(ac.Args["nonce"])
	if err != nil {
		return "", err
	}
	payload, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	if err != nil {
		return "", fmt.Errorf("dsl: data: %w", err)
	}
	if value == nil {
		value = new(big.Int)
	}
	return w.SendLegacyTx(ctx, accounts.LegacyTxArgs{ToHex: to, Value: value, Data: payload, Gas: gas, GasPrice: gasPrice, Nonce: nonce, AccessList: list})
}

func localAccessList(deps *interp.Deps, raw any) ([]accounts.AccessTuple, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var list []accounts.AccessTuple
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&list); err != nil {
		return nil, fmt.Errorf("dsl: accessList: %w", err)
	}
	if list == nil {
		return nil, fmt.Errorf("dsl: accessList must be an array")
	}
	for i := range list {
		if list[i].StorageKeys == nil {
			return nil, fmt.Errorf("dsl: accessList storageKeys must be an array")
		}
		address, err := ResolveAddress(deps, list[i].Address)
		if err != nil {
			return nil, err
		}
		list[i].Address = address
	}
	return list, nil
}

// sendDynamicAccessList preserves the supplied list during gas estimation and signing.
func sendDynamicAccessList(ctx context.Context, ac *interp.ActionCtx, w accounts.Wallet, to string, value *big.Int, data, rpcURL string) (string, error) {
	list, err := localAccessList(ac.Deps, ac.Args["accessList"])
	if err != nil {
		return "", err
	}
	gas, explicit := hexQuantity(ac.Args["gas"])
	if !explicit {
		message := rpc.SendTxArgs{From: w.Address(), To: to, Data: data, AccessList: list}
		if value != nil {
			message.Value = "0x" + value.Text(16)
		}
		if err := applyFeeArgs(&message, ac.Args); err != nil {
			return "", err
		}
		client, err := clientFor(ac.Deps, rpcURL)
		if err != nil {
			return "", err
		}
		if err := client.Call(ctx, "eth_estimateGas", &gas, message); err != nil {
			return "", err
		}
	}
	return sendWithAccessListGas(ctx, ac, w, to, value, data, gas, rpcURL, list)
}
