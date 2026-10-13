package testhelper

import (
	"context"
	"fmt"
	"math/big"
	"strings"

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
	gas, explicit := hexQuantity(ac.Args["gas"])
	if !explicit {
		message := map[string]any{"from": sender.Address, "to": to, "value": "0x" + value.Text(16)}
		if data != "" {
			message["data"] = data
		}
		if err := client.Call(ctx, "eth_estimateGas", &gas, message); err != nil {
			return err
		}
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

func amountFromQuantity(value string) (*big.Int, error) {
	amount, ok := new(big.Int).SetString(strings.TrimPrefix(value, "0x"), 16)
	if !ok || amount.Sign() < 0 {
		return nil, fmt.Errorf("dsl: invalid asset amount")
	}
	return amount, nil
}
