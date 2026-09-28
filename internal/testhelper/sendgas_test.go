package testhelper

import (
	"context"
	"math/big"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// gasWallet records the dynamic-fee transaction it was asked to sign.
type gasWallet struct {
	accounts.Wallet
	got      *accounts.DynamicTxArgs
	sentCoin bool
}

func (w *gasWallet) SendDynamicFeeTx(_ context.Context, a accounts.DynamicTxArgs) (string, error) {
	w.got = &a
	return "0xh", nil
}

func (w *gasWallet) SendCoin(context.Context, string, *big.Int) (string, error) {
	w.sentCoin = true
	return "0xh", nil
}

type gasProvider struct {
	accounts.AccountProvider
	w *gasWallet
}

func (p *gasProvider) OpenWallet(context.Context, []byte, string) (accounts.Wallet, error) {
	return p.w, nil
}

// TestSendTx_LocalKeyHonoursTheGasTheCaseAsks: a locally signed send with an
// explicit "gas" goes out with that limit. It used to go through SendCoin,
// which fixes 21000 — a case that sent 100000 gas to a reverting contract ran
// out of gas instead of reverting.
func TestSendTx_LocalKeyHonoursTheGasTheCaseAsks(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"eth_maxPriorityFeePerGas": "0x3b9aca00",
		"eth_getBlockByNumber":     map[string]any{"number": "0x5", "hash": "0xb", "baseFeePerGas": "0x77359400", "timestamp": "0x1", "transactions": []any{}},
	})
	w := &gasWallet{}
	d := deps()
	d.Accounts = &gasProvider{w: w}
	act, _ := d.Actions.Action(actionSendTx)
	err := act.Do(context.Background(), &interp.ActionCtx{Env: envWithNode(t, srv.URL), Deps: &d, Args: map[string]any{
		"key": "0x" + "11" + "00000000000000000000000000000000000000000000000000000000000001"[:62],
		"to":  "0x00000000000000000000000000000000000000aa", "gas": "100000", "data": "0x", "wait": false,
	}})
	if err != nil {
		t.Fatalf("sendTx: %v", err)
	}
	if w.sentCoin || w.got == nil {
		t.Fatalf("the send did not carry the gas limit (SendCoin used: %v)", w.sentCoin)
	}
	if w.got.Gas != 100000 {
		t.Errorf("gas = %d, want 100000", w.got.Gas)
	}
	// Suggested fees: tip 1 gwei, cap = 2 * base(2 gwei) + tip = 5 gwei.
	if w.got.GasTipCap.Int64() != 1_000_000_000 || w.got.GasFeeCap.Int64() != 5_000_000_000 {
		t.Errorf("fees = tip %v cap %v, want 1e9 and 5e9", w.got.GasTipCap, w.got.GasFeeCap)
	}
}
