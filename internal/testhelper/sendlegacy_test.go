package testhelper

import (
	"context"
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// typeWallet records which signing path a locally signed send took.
type typeWallet struct {
	accounts.Wallet
	called      string
	gasPrice    *big.Int
	transaction *accounts.LegacyTxArgs
}

func (w *typeWallet) SendLegacyTx(_ context.Context, args accounts.LegacyTxArgs) (string, error) {
	w.called, w.gasPrice, w.transaction = "legacy", args.GasPrice, &args
	if args.AccessList != nil {
		w.called = "accessList"
	}
	return "0xh", nil
}

func (w *typeWallet) SendLegacyGas(_ context.Context, _ string, _, gasPrice *big.Int) (string, error) {
	w.called, w.gasPrice = "legacy", gasPrice
	return "0xh", nil
}

func (w *typeWallet) SendAccessListGas(_ context.Context, _ string, _, gasPrice *big.Int) (string, error) {
	w.called, w.gasPrice = "accessList", gasPrice
	return "0xh", nil
}

func (w *typeWallet) SendDynamicFeeTx(context.Context, accounts.DynamicTxArgs) (string, error) {
	w.called = "dynamic"
	return "0xh", nil
}

func (w *typeWallet) SendCoin(context.Context, string, *big.Int) (string, error) {
	w.called = "coin"
	return "0xh", nil
}

func (w *typeWallet) SendFeeDelegated(context.Context, []byte, string, *big.Int) (string, error) {
	w.called = "feeDelegated"
	return "0xh", nil
}

type typeProvider struct {
	accounts.AccountProvider
	w *typeWallet
}

func (p *typeProvider) OpenWallet(context.Context, []byte, string) (accounts.Wallet, error) {
	return p.w, nil
}

const (
	testLocalKey = "0x1100000000000000000000000000000000000000000000000000000000000001"
	testTo       = "0x00000000000000000000000000000000000000aa"
)

// TestSendTx_LocalKeyGasPriceSelectsTheTypeTheCaseNamed: a locally signed send
// with gasPrice is a legacy transaction, and with gasPrice and an empty access
// list an EIP-2930 one. It used to fall through to a 0x02 send and drop the
// gas price without a word, so a case written to test the legacy type tested
// the dynamic-fee one instead.
func TestSendTx_LocalKeyGasPriceSelectsTheTypeTheCaseNamed(t *testing.T) {
	cases := []struct {
		name       string
		args       map[string]any
		wantCalled string
	}{
		{"gasPrice alone is legacy", map[string]any{"gasPrice": "1000"}, "legacy"},
		{"gasPrice with an empty access list is 0x01", map[string]any{"gasPrice": "1000", "accessList": []any{}}, "accessList"},
		{"no fee field keeps the plain transfer", map[string]any{}, "coin"},
		{"empty list with dynamic fee caps", map[string]any{"accessList": []any{}, "gas": "21000", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1"}, "dynamic"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &typeWallet{}
			err := runLocalSend(t, w, tc.args)
			if err != nil {
				t.Fatalf("sendTx: %v", err)
			}
			if w.called != tc.wantCalled {
				t.Fatalf("signed as %q, want %q", w.called, tc.wantCalled)
			}
			if (tc.wantCalled == "legacy" || tc.wantCalled == "accessList") && w.gasPrice.Int64() != 1000 {
				t.Errorf("gasPrice = %v, want 1000", w.gasPrice)
			}
		})
	}
}

// TestSendTx_LocalKeyRefusesFieldsItWouldDrop: a combination the local path
// cannot sign is refused before anything is sent, rather than sent without the
// field the case wrote.
func TestSendTx_LocalKeyRefusesFieldsItWouldDrop(t *testing.T) {
	cases := []struct {
		name    string
		args    map[string]any
		wantErr string
	}{
		{"fee delegation with legacy price", map[string]any{"feePayerKey": testLocalKey, "gasPrice": "3"}, "gasPrice"},
		{"two fee payer sources", map[string]any{"feePayerKey": testLocalKey, "feePayer": "sponsor"}, "mutually exclusive"},
		{"empty fee payer", map[string]any{"feePayer": ""}, "feePayer"},
		{"empty fee key", map[string]any{"feePayerKey": ""}, "feePayerKey"},
		{"gasPrice with fee caps", map[string]any{"gasPrice": "1000", "maxFeePerGas": "2000", "maxPriorityFeePerGas": "1"}, "gasPrice"},
		{"non-array access list", map[string]any{"gasPrice": "1000", "accessList": map[string]any{"address": testTo}}, "accessList"},
		{"unknown access list field", map[string]any{"gasPrice": "1000", "accessList": []any{map[string]any{"address": testTo, "typo": true}}}, "accessList"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &typeWallet{}
			err := runLocalSend(t, w, tc.args)
			if err == nil {
				t.Fatalf("sendTx accepted %v and signed as %q", tc.args, w.called)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not name %q", err, tc.wantErr)
			}
			if w.called != "" {
				t.Errorf("something was signed (%q) before the refusal", w.called)
			}
		})
	}
}

func TestSendTx_LocalLegacyPreservesExplicitFields(t *testing.T) {
	w := &typeWallet{}
	key := "0x" + strings.Repeat("0", 63) + "1"
	err := runLocalSend(t, w, map[string]any{"gasPrice": "1000", "gas": "70000", "nonce": "7", "data": "0xabcdef", "accessList": []any{map[string]any{"address": testTo, "storageKeys": []any{key}}}})
	if err != nil {
		t.Fatal(err)
	}
	a := w.transaction
	if a == nil || a.Gas != 70000 || a.Nonce == nil || *a.Nonce != 7 || a.ToHex != testTo || a.Value.Cmp(big.NewInt(1)) != 0 || string(a.Data) != string([]byte{0xab, 0xcd, 0xef}) || a.GasPrice.Cmp(big.NewInt(1000)) != 0 || len(a.AccessList) != 1 || a.AccessList[0].Address != testTo || len(a.AccessList[0].StorageKeys) != 1 || a.AccessList[0].StorageKeys[0] != key {
		t.Fatalf("explicit transaction changed: %+v", a)
	}
}

func runLocalSend(t *testing.T, w *typeWallet, extra map[string]any) error {
	t.Helper()
	srv := mockRPC(t, map[string]any{})
	d := deps()
	d.Accounts = &typeProvider{w: w}
	act, _ := d.Actions.Action(actionSendTx)
	args := map[string]any{"key": testLocalKey, "to": testTo, "value": "1", "wait": false}
	for k, v := range extra {
		args[k] = v
	}
	return act.Do(context.Background(), &interp.ActionCtx{Env: envWithNode(t, srv.URL), Deps: &d, Args: args})
}
