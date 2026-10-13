package testhelper

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// localAssetTransaction preserves the asset action's own argument contract.
// Its payload is supplied by the action; sendTx's unrelated options are not
// accidentally added to faucet/registerContract or their generated UI forms.
func localAssetTransaction(ctx context.Context, ac *interp.ActionCtx, sender Account, to string, value *big.Int, data string) error {
	if ac.Deps == nil || ac.Deps.Accounts == nil {
		return fmt.Errorf("dsl: local asset needs an account provider")
	}
	url := selectorTarget(ac.Env, ac.Args)
	wallet, err := ac.Deps.Accounts.OpenWallet(ctx, sender.Key, url)
	if err != nil {
		return err
	}
	client, err := clientFor(ac.Deps, url)
	if err != nil {
		return err
	}
	if value == nil {
		value = new(big.Int)
	}
	gas, err := localAssetGas(ctx, ac, sender.Address, to, value, data)
	if err != nil {
		return err
	}
	hash, err := sendWithGas(ctx, ac, wallet, to, value, data, gas, url)
	if err != nil {
		return err
	}
	receipt, err := waitReceipt(ctx, client, hash, durationArg(ac.Args, "timeout", defaultTxTimeout), durationArg(ac.Args, "pollInterval", defaultTxPollInterval))
	if err != nil {
		return err
	}
	ac.Hash, ac.Receipt = hash, receipt
	if statusReverted(receipt) {
		return fmt.Errorf("dsl: local asset %s reverted", hash)
	}
	return nil
}

// localAssetGas estimates the actual call while retaining its fee model.
// Estimation failures do not replace explicit inputs with SDK defaults.
func localAssetGas(ctx context.Context, ac *interp.ActionCtx, from, to string, value *big.Int, data string) (string, error) {
	if raw, supplied := ac.Args["gas"]; supplied {
		gas, ok := hexQuantity(raw)
		if !ok {
			return "", fmt.Errorf("dsl: local asset gas must be a quantity")
		}
		return gas, nil
	}
	message := rpc.SendTxArgs{From: from, To: to, Data: data}
	if value != nil {
		message.Value = "0x" + value.Text(16)
	}
	if err := applyFeeArgs(&message, ac.Args); err != nil {
		return "", err
	}
	client, err := clientFor(ac.Deps, selectorTarget(ac.Env, ac.Args))
	if err != nil {
		return "", err
	}
	var gas string
	if err := client.Call(ctx, "eth_estimateGas", &gas, message); err != nil {
		return "", fmt.Errorf("dsl: local asset estimate gas: %w", err)
	}
	return gas, nil
}

func amountFromQuantity(value string) (*big.Int, error) {
	amount, ok := new(big.Int).SetString(strings.TrimPrefix(value, "0x"), 16)
	if !ok || amount.Sign() < 0 {
		return nil, fmt.Errorf("dsl: invalid asset amount")
	}
	return amount, nil
}
