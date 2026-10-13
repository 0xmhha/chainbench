package accounts_test

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

type refusingWalletRPC struct{}

func (refusingWalletRPC) Call(_ context.Context, method string, result any, _ ...any) error {
	if method == "eth_sendRawTransaction" {
		return errors.New("test refusal")
	}
	value := `"0x1"`
	if method == "eth_getProof" {
		value = `{"extra":"0x0"}`
	}
	return json.Unmarshal([]byte(value), result)
}

func TestCallerOwnedFeeDelegationRefusalReturnsNoTransactionHash(t *testing.T) {
	provider, err := accounts.ForChainWithRPC("stablenet", func(string) accounts.RPCRequester { return refusingWalletRPC{} })
	if err != nil {
		t.Fatal(err)
	}
	wallet, err := provider.OpenWallet(context.Background(), mustKey(t, testPrivKeyHex), "unused")
	if err != nil {
		t.Fatal(err)
	}
	hash, err := wallet.SendFeeDelegated(context.Background(), mustKey(t, strings.Repeat("0", 63)+"2"), testAddress, big.NewInt(1))
	if err == nil || hash != "" {
		t.Fatalf("refused submission must have no hash: %q %v", hash, err)
	}
}

func TestEveryWalletOperationRetainsCallerAuthentication(t *testing.T) {
	t.Setenv("CHAINBENCH_TEST_WALLET_TOKEN", "wallet-transport-test-token")
	key := mustKey(t, testPrivKeyHex)
	authority := mustKey(t, strings.Repeat("0", 63)+"2")
	const recipient = "0x000000000000000000000000000000000000dead"
	ctx := context.Background()
	for _, operation := range []struct {
		name     string
		typeByte string
		invoke   func(accounts.AccountProvider, accounts.Wallet, string) error
	}{
		{"faucet", "02", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := p.Faucet(ctx, key, recipient, big.NewInt(1), url)
			return err
		}},
		{"coin", "02", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := w.SendCoin(ctx, recipient, big.NewInt(1))
			return err
		}},
		{"deploy", "02", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, _, err := w.Deploy(ctx, []byte{0x60, 0x00}, nil)
			return err
		}},
		{"execute", "02", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := w.Execute(ctx, recipient, []byte{0xab, 0xcd}, nil)
			return err
		}},
		{"delegated", "16", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := w.SendFeeDelegated(ctx, authority, recipient, big.NewInt(1))
			return err
		}},
		{"legacy", "", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := w.SendLegacy(ctx, recipient, big.NewInt(1))
			return err
		}},
		{"dynamic", "02", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := w.SendDynamicFee(ctx, recipient, big.NewInt(1))
			return err
		}},
		{"access-list", "01", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := w.SendAccessList(ctx, recipient, big.NewInt(1))
			return err
		}},
		{"dynamic-fees", "02", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := w.SendDynamicFeeGas(ctx, recipient, big.NewInt(1), big.NewInt(3), big.NewInt(1))
			return err
		}},
		{"legacy-price", "", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := w.SendLegacyGas(ctx, recipient, big.NewInt(1), big.NewInt(3))
			return err
		}},
		{"access-price", "01", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := w.SendAccessListGas(ctx, recipient, big.NewInt(1), big.NewInt(3))
			return err
		}},
		{"specified", "02", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			nonce := uint64(7)
			_, err := w.SendDynamicFeeTx(ctx, accounts.DynamicTxArgs{ToHex: recipient, Value: big.NewInt(1), Data: []byte{0xab}, Gas: 45000, Nonce: &nonce, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(3)})
			return err
		}},
		{"set-code", "04", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			_, err := w.SendSetCode(ctx, authority, recipient)
			return err
		}},
		{"authorization", "none", func(p accounts.AccountProvider, w accounts.Wallet, url string) error {
			auth, err := w.SignAuthorization(ctx, authority, recipient)
			if err == nil && (auth["address"] != recipient || auth["chainId"] != "0x205b" || auth["nonce"] != "0x3") {
				t.Errorf("authorization lost fields: %+v", auth)
			}
			return err
		}},
	} {
		t.Run(operation.name, func(t *testing.T) {
			raw := ""
			server := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, request *http.Request) {
				if request.Header.Get("X-Test-Key") != "wallet-transport-test-token" {
					t.Error("wallet bypassed its caller-owned transport")
					http.Error(out, "unauthorized", http.StatusUnauthorized)
					return
				}
				var body struct {
					ID     any               `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				var result any = "0x1"
				switch body.Method {
				case "eth_chainId":
					result = "0x205b"
				case "eth_getTransactionCount":
					result = "0x3"
				case "eth_estimateGas":
					result = "0x10000"
				case "eth_getProof":
					result = map[string]string{"extra": "0x0"}
				case "eth_sendRawTransaction":
					if err := json.Unmarshal(body.Params[0], &raw); err != nil {
						t.Error(err)
					}
					result = "0x" + strings.Repeat("ab", 32)
				case "eth_sendTransaction":
					t.Error("local wallet delegated signing to a node")
				}
				_ = json.NewEncoder(out).Encode(map[string]any{"jsonrpc": "2.0", "id": body.ID, "result": result})
			}))
			defer server.Close()
			client, err := remote.HTTPClientFromAuth(remote.Auth{"type": "api-key", "env": "CHAINBENCH_TEST_WALLET_TOKEN", "header": "X-Test-Key"}, func(name string) string {
				if name == "CHAINBENCH_TEST_WALLET_TOKEN" {
					return "wallet-transport-test-token"
				}
				return ""
			})
			if err != nil {
				t.Fatal(err)
			}
			provider, err := accounts.ForChainWithRPC("stablenet", func(url string) accounts.RPCRequester { return rpc.DialWithClient(url, client) })
			if err != nil {
				t.Fatal(err)
			}
			wallet, err := provider.OpenWallet(ctx, key, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			if err := operation.invoke(provider, wallet, server.URL); err != nil {
				t.Fatal(err)
			}
			if operation.typeByte == "none" {
				if raw != "" {
					t.Fatal("read-only authorization submitted a transaction")
				}
			} else if raw == "" || operation.typeByte != "" && !strings.HasPrefix(raw, "0x"+operation.typeByte) {
				t.Fatalf("wrong signed envelope: %q", raw)
			}
		})
	}
}
