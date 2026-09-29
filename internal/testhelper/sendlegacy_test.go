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
	called   string
	gasPrice *big.Int
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
			if tc.wantCalled != "coin" && w.gasPrice.Int64() != 1000 {
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
		{"gasPrice with fee caps", map[string]any{"gasPrice": "1000", "maxFeePerGas": "2000", "maxPriorityFeePerGas": "1"}, "gasPrice"},
		{"gasPrice with gas", map[string]any{"gasPrice": "1000", "gas": "30000"}, "gas"},
		{"gasPrice with data", map[string]any{"gasPrice": "1000", "data": "0x01"}, "data"},
		{"gasPrice with nonce", map[string]any{"gasPrice": "1000", "nonce": "0"}, "nonce"},
		{"access list without gasPrice", map[string]any{"accessList": []any{}}, "accessList"},
		{"non-empty access list", map[string]any{"gasPrice": "1000", "accessList": []any{map[string]any{"address": testTo}}}, "accessList"},
		{"feePayerKey with gas", map[string]any{"feePayerKey": testLocalKey, "gas": "30000"}, "feePayerKey"},
		{"feePayerKey with data", map[string]any{"feePayerKey": testLocalKey, "data": "0x01"}, "feePayerKey"},
		{"feePayerKey with nonce", map[string]any{"feePayerKey": testLocalKey, "nonce": "0"}, "feePayerKey"},
		{"feePayerKey with fee caps", map[string]any{"feePayerKey": testLocalKey, "maxFeePerGas": "2", "maxPriorityFeePerGas": "1"}, "feePayerKey"},
		{"feePayerKey with access list", map[string]any{"feePayerKey": testLocalKey, "accessList": []any{}}, "feePayerKey"},
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
