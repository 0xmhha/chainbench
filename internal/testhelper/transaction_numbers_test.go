package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type numericWallet struct {
	accounts.Wallet
	sends atomic.Int32
}

func (w *numericWallet) Address() string { return testTo }
func (w *numericWallet) SendCoin(context.Context, string, *big.Int) (string, error) {
	w.sends.Add(1)
	return "0xh", nil
}
func (w *numericWallet) Execute(context.Context, string, []byte, *big.Int) (string, error) {
	w.sends.Add(1)
	return "0xh", nil
}
func (w *numericWallet) Deploy(context.Context, []byte, *big.Int) (string, string, error) {
	w.sends.Add(1)
	return "0xh", testTo, nil
}
func (w *numericWallet) SendDynamicFeeTx(context.Context, accounts.DynamicTxArgs) (string, error) {
	w.sends.Add(1)
	return "0xh", nil
}
func (w *numericWallet) SendLegacyTx(context.Context, accounts.LegacyTxArgs) (string, error) {
	w.sends.Add(1)
	return "0xh", nil
}

type numericProvider struct {
	accounts.AccountProvider
	w     *numericWallet
	opens atomic.Int32
}

func (p *numericProvider) OpenWallet(context.Context, []byte, string) (accounts.Wallet, error) {
	p.opens.Add(1)
	return p.w, nil
}

func TestTransactionActionsRejectInvalidNumbersBeforeRPCOrSigning(t *testing.T) {
	for _, mode := range []string{"node", "local"} {
		d, ring := depsWithRing(t)
		d.Actions = testhelperRegistry()
		for _, action := range []string{"sendTx", "faucet", "deployContract", "registerContract", "load"} {
			fields := []string{"gas", "nonce", "gasPrice", "maxFeePerGas", "maxPriorityFeePerGas"}
			if action == "faucet" {
				fields = append(fields, "amount")
			} else if action != "load" {
				fields = append(fields, "value")
			} else {
				fields = append(fields, "blocks", "fillPercent")
			}
			for _, field := range fields {
				badValues := []any{"not-a-number", float64(1.5), -1, true, nil, float64(9007199254740992)}
				if field == "blocks" || field == "fillPercent" {
					badValues = append(badValues, 0, "18446744073709551615")
				}
				if field == "fillPercent" {
					badValues = append(badValues, 101)
				}
				for i, bad := range badValues {
					t.Run(fmt.Sprintf("%s/%s/%s/%d", mode, action, field, i), func(t *testing.T) {
						var requests atomic.Int32
						server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
							requests.Add(1)
							var req struct {
								ID     any    `json:"id"`
								Method string `json:"method"`
							}
							_ = json.NewDecoder(r.Body).Decode(&req)
							var result any = "0x1"
							switch req.Method {
							case "eth_sendTransaction":
								result = "0xh"
							case "eth_getTransactionReceipt":
								result = map[string]any{"status": "0x1", "contractAddress": testTo}
							case "eth_getBlockByNumber":
								result = map[string]any{"baseFeePerGas": "0x1", "gasLimit": "0xf4240", "number": "0x1"}
							case "eth_estimateGas":
								result = "0x11170"
							}
							_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
						}))
						defer server.Close()
						wallet := &numericWallet{}
						provider := &numericProvider{w: wallet}
						local := *d
						local.RPC = deps().RPC
						local.Accounts = provider
						local.Keys = ring
						args := map[string]any{"from": testTo, "to": testTo, "amount": "1", "value": "1", "gas": "300000", "bytecode": "0x60006000f3", "data": "0x01", "wait": false, "blocks": 1}
						if mode == "local" {
							args["from"] = "dev1"
						}
						args[field] = bad
						act, _ := local.Actions.Action(action)
						ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
						defer cancel()
						err := act.Do(ctx, &interp.ActionCtx{Env: envWithNode(t, server.URL), Deps: &local, Args: args})
						if err == nil || !strings.Contains(err.Error(), field) {
							t.Errorf("invalid %s accepted or not identified: %v", field, err)
						}
						if requests.Load() != 0 || provider.opens.Load() != 0 || wallet.sends.Load() != 0 {
							t.Errorf("invalid number reached RPC/wallet: requests=%d opens=%d sends=%d", requests.Load(), provider.opens.Load(), wallet.sends.Load())
						}
					})
				}
			}
		}
	}
}

func TestQuantityConversionIsExactAndCannotPanic(t *testing.T) {
	for _, bad := range []any{math.NaN(), math.Inf(1), math.Inf(-1), 1.5, float64(9007199254740992), "0xgg", "-1", true, nil} {
		t.Run(fmt.Sprint(bad), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("quantity conversion panicked: %v", r)
				}
			}()
			if quantity, ok := hexQuantity(bad); ok {
				t.Fatalf("invalid quantity converted to %q", quantity)
			}
		})
	}
	for _, c := range []struct {
		input any
		want  string
	}{
		{uint64(math.MaxUint64), "0xffffffffffffffff"}, {json.Number("18446744073709551615"), "0xffffffffffffffff"}, {int64(7), "0x7"}, {"0X000A", "0xa"}, {" 00010 ", "0xa"}, {float64(9007199254740991), "0x1fffffffffffff"},
	} {
		if got, ok := hexQuantity(c.input); !ok || got != c.want {
			t.Errorf("exact conversion of %v: %q %v, want %q", c.input, got, ok, c.want)
		}
	}
}

func TestControlQuantityConversionIsExact(t *testing.T) {
	for _, bad := range []any{1.5, math.NaN(), math.Inf(1), float64(9007199254740992), "18446744073709551616", -1, nil, true} {
		if value, ok := uintArg(bad); ok {
			t.Errorf("invalid control %v converted to %d", bad, value)
		}
	}
	for _, input := range []any{uint64(math.MaxUint64), json.Number("18446744073709551615"), "0XFFFFFFFFFFFFFFFF", "18446744073709551615"} {
		if value, ok := uintArg(input); !ok || value != math.MaxUint64 {
			t.Errorf("exact control %v: %d %v", input, value, ok)
		}
	}
}

func TestLoadControlBoundsAndDefaults(t *testing.T) {
	for _, args := range []map[string]any{
		{}, {"blocks": 1, "fillPercent": 100}, {"blocks": "0X02", "fillPercent": "0x64"},
		{"blocks": uint64(^uint(0) >> 1), "fillPercent": "0001"},
	} {
		if err := validateLoadControls(args); err != nil {
			t.Errorf("valid controls %v: %v", args, err)
		}
	}
	for _, args := range []map[string]any{
		{"blocks": uint64(^uint(0)>>1) + 1}, {"blocks": "0x0"}, {"fillPercent": "0X65"},
	} {
		if err := validateLoadControls(args); err == nil {
			t.Errorf("invalid controls accepted: %v", args)
		}
	}
}
