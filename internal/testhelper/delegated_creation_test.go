package testhelper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type creationProvider struct{ *numericProvider }

func (p *creationProvider) AddressForKey([]byte) (string, error) { return testTo, nil }

func TestDeployContractRejectsInvalidDelegationBeforeRPCOrOpeningWallet(t *testing.T) {
	for _, mode := range []string{"local", "node"} {
		invalid := map[string]map[string]any{
			"mutually exclusive": {"feePayer": "dev2", "feePayerKey": testLocalKey},
			"gasPrice":           {"feePayer": "dev2", "gasPrice": "3"},
			"feePayer":           {"feePayer": nil},
			"come together":      {"feePayer": "dev1", "maxFeePerGas": "3"},
			"feePayerKey":        {"feePayerKey": ""},
		}
		if mode == "node" {
			invalid["local signing account"] = map[string]any{"feePayer": "dev2"}
		}
		for name, bad := range invalid {
			t.Run(mode+"/"+name, func(t *testing.T) {
				var calls atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					var q struct {
						ID     any
						Method string
					}
					_ = json.NewDecoder(r.Body).Decode(&q)
					result := any("0xh")
					if q.Method == "eth_estimateGas" {
						result = "0x10000"
					}
					if q.Method == "eth_getTransactionReceipt" {
						result = map[string]any{"status": "0x1", "contractAddress": testTo}
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
				}))
				defer srv.Close()
				d, ring := depsWithRing(t)
				w := &numericWallet{}
				provider := &creationProvider{numericProvider: &numericProvider{w: w}}
				d.Accounts = provider
				d.RPC = deps().RPC
				d.Keys = ring
				args := map[string]any{"from": testTo, "bytecode": "0x60006000f3", "gas": "70000", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1"}
				if mode == "local" {
					args["from"] = "dev1"
				}
				if name == "come together" {
					delete(args, "maxPriorityFeePerGas")
					delete(args, "gas")
				}
				for k, v := range bad {
					args[k] = v
				}
				err := (deployContractAction{}).Do(context.Background(), &interp.ActionCtx{Env: envWithNode(t, srv.URL), Deps: d, Args: args})
				if err == nil || !strings.Contains(err.Error(), name) {
					t.Errorf("invalid delegation not identified: %v", err)
				}
				if calls.Load() != 0 || provider.opens.Load() != 0 || w.sends.Load() != 0 {
					t.Errorf("invalid delegation reached RPC/wallet: calls=%d opens=%d sends=%d", calls.Load(), provider.opens.Load(), w.sends.Load())
				}
			})
		}
	}
}

func TestDelegatedCreationArgumentsAreConsumed(t *testing.T) {
	s := argumentCase(t, `{"do":"deployContract","from":"dev1","feePayer":"dev2","bytecode":"0x60006000f3","accessList":[],"gas":"70000"},{"expect":"chainId","is":"1"}`)
	if got := IgnoredArguments(s); len(got) != 0 {
		t.Fatalf("delegated creation must consume its inputs: %v", got)
	}
	s = argumentCase(t, `{"do":"newAccount","saveKey":"k"},{"do":"deployContract","key":"$k","feePayerKey":"$k","bytecode":"0x60006000f3"},{"expect":"chainId","is":"1"}`)
	if got := IgnoredArguments(s); len(got) != 0 {
		t.Fatalf("key-bound delegated creation must consume its inputs: %v", got)
	}
}
