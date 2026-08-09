package testspec

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// Asset and contract action names (design §3.2, F11 AC-6/7).
const (
	actionFaucet           = "faucet"
	actionDeployContract   = "deployContract"
	actionRegisterContract = "registerContract"
)

// seedAssetBuiltins registers the funding and contract actions.
func seedAssetBuiltins(r Registry) {
	r.RegisterAction(actionFaucet, faucetAction{})
	r.RegisterAction(actionDeployContract, deployContractAction{})
	r.RegisterAction(actionRegisterContract, registerContractAction{})
}

// faucetAction funds an address with gas money. A signing key generated at run
// time is not in the genesis alloc, so it starts with a zero balance and cannot
// pay for its own first transaction; faucet is how a spec tops it up before
// using it (F11 AC-6).
//
// Args: to (required), amount (required, decimal or 0x-hex wei), from (funder;
// defaults to the target node's unlocked coinbase), on, gas, timeout,
// pollInterval.
type faucetAction struct{}

func (faucetAction) Do(ctx context.Context, ac *ActionCtx) error {
	to, _ := ac.Args["to"].(string)
	if to == "" {
		return fmt.Errorf("testspec: faucet requires \"to\" (the address to fund)")
	}
	value, ok := hexQuantity(ac.Args["amount"])
	if !ok {
		return fmt.Errorf("testspec: faucet requires a numeric \"amount\" in wei")
	}
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	from, err := funder(ctx, c, ac.Args)
	if err != nil {
		return err
	}

	args := rpc.SendTxArgs{From: from, To: to, Value: value}
	if g, ok := hexQuantity(ac.Args["gas"]); ok {
		args.Gas = g
	}
	if err := applyFeeArgs(&args, ac.Args); err != nil {
		return err
	}
	receipt, hash, err := sendAndConfirm(ctx, c, args, ac.Args)
	if err != nil {
		return fmt.Errorf("testspec: faucet: %w", err)
	}
	ac.Hash, ac.Receipt = hash, receipt
	if statusReverted(receipt) {
		return fmt.Errorf("testspec: faucet %s -> %s reverted", from, to)
	}
	return nil
}

// deployContractAction deploys bytecode and binds the deployed address, so a
// later step can call the contract it just created (F11 AC-7). A deployment is a
// transaction with no "to"; the address only appears in the receipt, which is
// why this cannot be a plain sendTx.
//
// Args: bytecode (required), from (defaults to the node coinbase), on, gas,
// value, save, timeout, pollInterval.
type deployContractAction struct{}

func (deployContractAction) Do(ctx context.Context, ac *ActionCtx) error {
	code, _ := ac.Args["bytecode"].(string)
	if code == "" {
		code, _ = ac.Args["data"].(string)
	}
	if code == "" {
		return fmt.Errorf("testspec: deployContract requires \"bytecode\"")
	}
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	from, err := funder(ctx, c, ac.Args)
	if err != nil {
		return err
	}

	args := rpc.SendTxArgs{From: from, Data: code}
	if g, ok := hexQuantity(ac.Args["gas"]); ok {
		args.Gas = g
	}
	if v, ok := hexQuantity(ac.Args["value"]); ok {
		args.Value = v
	}
	if err := applyFeeArgs(&args, ac.Args); err != nil {
		return err
	}
	receipt, hash, err := sendAndConfirm(ctx, c, args, ac.Args)
	if err != nil {
		return fmt.Errorf("testspec: deployContract: %w", err)
	}
	ac.Hash, ac.Receipt = hash, receipt
	if statusReverted(receipt) {
		return fmt.Errorf("testspec: deployContract %s reverted", hash)
	}
	addr, _ := receipt["contractAddress"].(string)
	if addr == "" {
		return fmt.Errorf("testspec: deployContract %s: receipt carries no contract address", hash)
	}
	ac.Value = addr
	return nil
}

// registerContractAction calls a deployed contract to register something with it
// — the node-registration half of F11 AC-7, where a bp announces itself to a
// contract the spec just deployed.
//
// It is deliberately a thin named form of sendTx: the difference is that "to" is
// required (a registration always has a target, and a missing "to" would
// silently become a second deployment) and a revert is always a failure. The
// name is what a ported case reads as, so specs say what they mean.
//
// Args: to (required, typically "$addr" from deployContract), data (required),
// from, on, gas, value, timeout, pollInterval.
type registerContractAction struct{}

func (registerContractAction) Do(ctx context.Context, ac *ActionCtx) error {
	to, _ := ac.Args["to"].(string)
	if to == "" {
		return fmt.Errorf("testspec: registerContract requires \"to\" (the deployed contract address)")
	}
	data, _ := ac.Args["data"].(string)
	if data == "" {
		return fmt.Errorf("testspec: registerContract requires \"data\" (the encoded call)")
	}
	c, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return err
	}
	from, err := funder(ctx, c, ac.Args)
	if err != nil {
		return err
	}

	args := rpc.SendTxArgs{From: from, To: to, Data: data}
	if g, ok := hexQuantity(ac.Args["gas"]); ok {
		args.Gas = g
	}
	if v, ok := hexQuantity(ac.Args["value"]); ok {
		args.Value = v
	}
	if err := applyFeeArgs(&args, ac.Args); err != nil {
		return err
	}
	receipt, hash, err := sendAndConfirm(ctx, c, args, ac.Args)
	if err != nil {
		return fmt.Errorf("testspec: registerContract: %w", err)
	}
	ac.Hash, ac.Receipt = hash, receipt
	if statusReverted(receipt) {
		return fmt.Errorf("testspec: registerContract %s reverted", hash)
	}
	return nil
}

// funder resolves the sending account: the explicit "from" if given, else the
// target node's unlocked coinbase (a validator signs with it, so it is the one
// account a locally launched network can always spend from).
func funder(ctx context.Context, c *rpc.Client, args map[string]any) (string, error) {
	if from, ok := args["from"].(string); ok && from != "" {
		return from, nil
	}
	from, err := c.Coinbase(ctx)
	if err != nil {
		return "", fmt.Errorf("no \"from\" given and the node's coinbase is unavailable: %w", err)
	}
	if from == "" {
		return "", fmt.Errorf("no \"from\" given and the node reports no coinbase")
	}
	return from, nil
}

// sendAndConfirm submits a transaction and waits for its receipt, using the
// action's timeout/pollInterval overrides.
func sendAndConfirm(ctx context.Context, c *rpc.Client, args rpc.SendTxArgs, opts map[string]any) (map[string]any, string, error) {
	hash, err := c.SendTransaction(ctx, args)
	if err != nil {
		return nil, "", err
	}
	receipt, err := waitReceipt(ctx, c, hash,
		durationArg(opts, "timeout", defaultTxTimeout),
		durationArg(opts, "pollInterval", defaultTxPollInterval))
	if err != nil {
		return nil, hash, err
	}
	return receipt, hash, nil
}

// applyFeeArgs copies a step's fee and nonce arguments onto a transaction.
//
// Fee-policy cases need to set the caps deliberately (a maxFeePerGas below the
// base fee must be rejected), and nonce cases need to pin the position (submit
// out of order, or replace a pending transaction at the same nonce). Both are
// meaningless to a receipt-only view, which is why they belong on the send.
//
// The two fee forms are mutually exclusive: a node rejects a transaction that
// carries both, so catching it here names the mistake instead of surfacing an
// opaque RPC error.
func applyFeeArgs(args *rpc.SendTxArgs, in map[string]any) error {
	gasPrice, hasLegacy := hexQuantity(in["gasPrice"])
	maxFee, hasMaxFee := hexQuantity(in["maxFeePerGas"])
	tip, hasTip := hexQuantity(in["maxPriorityFeePerGas"])

	if hasLegacy && (hasMaxFee || hasTip) {
		return fmt.Errorf("testspec: sendTx: \"gasPrice\" and \"maxFeePerGas\"/\"maxPriorityFeePerGas\" are mutually exclusive")
	}
	if hasLegacy {
		args.GasPrice = gasPrice
	}
	if hasMaxFee {
		args.MaxFeePerGas = maxFee
	}
	if hasTip {
		args.MaxPriorityFeePerGas = tip
	}
	if nonce, ok := hexQuantity(in["nonce"]); ok {
		args.Nonce = nonce
	}
	return nil
}
