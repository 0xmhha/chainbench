package testengine_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	sdkcrypto "github.com/0xmhha/accounts/crypto"
	sdktx "github.com/0xmhha/accounts/tx"
	sdktypes "github.com/0xmhha/accounts/types"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestDeclaredExternalAccountSignsAssetActionsLocally(t *testing.T) {
	for _, sample := range []struct {
		name         string
		step         map[string]any
		transactions int32
	}{
		{"dynamic-auto-full", map[string]any{"do": "sendTx", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1"}, 1},
		{"dynamic-auto-nonce", map[string]any{"do": "sendTx", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "nonce": "7"}, 1},
		{"dynamic-auto-fees", map[string]any{"do": "sendTx", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1"}, 1},
		{"legacy-auto", map[string]any{"do": "sendTx", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "nonce": "7", "gasPrice": "3"}, 1},
		{"faucet", map[string]any{"do": "faucet", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "amount": "1"}, 1},
		{"fee-full", map[string]any{"do": "sendTx", "from": "payer", "feePayer": "sponsor", "to": "0x000000000000000000000000000000000000dead", "value": "5", "gas": "70000", "data": "0xabcdef", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1", "accessList": []any{map[string]any{"address": "0x000000000000000000000000000000000000dead", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
		{"fee-auto", map[string]any{"do": "sendTx", "from": "payer", "feePayer": "sponsor", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1", "accessList": []any{map[string]any{"address": "0x000000000000000000000000000000000000dead", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
		{"fee-create-full", map[string]any{"do": "deployContract", "from": "payer", "feePayer": "sponsor", "bytecode": "0x60006000f3", "value": "5", "gas": "70000", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1", "accessList": []any{map[string]any{"address": "0x000000000000000000000000000000000000dead", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
		{"fee-create-auto", map[string]any{"do": "deployContract", "from": "payer", "feePayer": "sponsor", "bytecode": "0x60006000f3", "value": "5", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1", "accessList": []any{map[string]any{"address": "0x000000000000000000000000000000000000dead", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
		{"fee-create-default", map[string]any{"do": "deployContract", "from": "payer", "feePayer": "sponsor", "bytecode": "0x60006000f3", "value": "5"}, 1},
		{"fee-create-refused", map[string]any{"do": "deployContract", "from": "payer", "feePayer": "sponsor", "bytecode": "0x60006000f3", "value": "5"}, 0},
		{"dynamic-create-list", map[string]any{"do": "deployContract", "from": "payer", "bytecode": "0x60006000f3", "value": "5", "gas": "70000", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1", "accessList": []any{map[string]any{"address": "0x000000000000000000000000000000000000dead", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
		{"access-list-create", map[string]any{"do": "deployContract", "from": "payer", "bytecode": "0x60006000f3", "value": "5", "gas": "70000", "nonce": "7", "gasPrice": "3", "accessList": []any{map[string]any{"address": "0x000000000000000000000000000000000000dead", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
		{"fee-create-key", map[string]any{"do": "deployContract", "from": "payer", "feePayerKey": strings.Repeat("0", 63) + "2", "bytecode": "0x60006000f3", "value": "5", "gas": "70000", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1", "accessList": []any{map[string]any{"address": "0x000000000000000000000000000000000000dead", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
		{"deploy", map[string]any{"do": "deployContract", "from": "payer", "bytecode": "0x60006000f3", "gas": "200000", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1"}, 1},
		{"deploy-auto-dynamic", map[string]any{"do": "deployContract", "from": "payer", "bytecode": "0x60006000f3", "value": "5", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1"}, 1},
		{"deploy-auto-legacy", map[string]any{"do": "deployContract", "from": "payer", "bytecode": "0x60006000f3", "value": "5", "nonce": "7", "gasPrice": "3"}, 1},
		{"deploy-auto-nonce", map[string]any{"do": "deployContract", "from": "payer", "bytecode": "0x60006000f3", "value": "5", "nonce": "7"}, 1},
		{"register", map[string]any{"do": "registerContract", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "data": "0xabcdef", "gas": "200000", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1"}, 1},
		{"load", map[string]any{"do": "load", "from": "payer", "gas": "300000", "blocks": 2, "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1"}, 2},
		{"legacy", map[string]any{"do": "sendTx", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "gas": "70000", "nonce": "7", "gasPrice": "3"}, 1},
		{"faucet-legacy", map[string]any{"do": "faucet", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "amount": "5", "gas": "70000", "nonce": "7", "gasPrice": "3"}, 1},
		{"deploy-legacy", map[string]any{"do": "deployContract", "from": "payer", "bytecode": "0x60006000f3", "value": "5", "gas": "70000", "nonce": "7", "gasPrice": "3"}, 1},
		{"register-legacy", map[string]any{"do": "registerContract", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "gas": "70000", "nonce": "7", "gasPrice": "3"}, 1},
		{"load-legacy", map[string]any{"do": "load", "from": "payer", "gas": "300000", "blocks": 2, "nonce": "7", "gasPrice": "3"}, 2},
		{"access-list", map[string]any{"do": "sendTx", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "gas": "70000", "nonce": "7", "gasPrice": "3", "accessList": []any{map[string]any{"address": "0x000000000000000000000000000000000000dead", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
		{"dynamic-list", map[string]any{"do": "sendTx", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "gas": "70000", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1", "accessList": []any{map[string]any{"address": "0x000000000000000000000000000000000000dead", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
		{"dynamic-list-label", map[string]any{"do": "sendTx", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "gas": "70000", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1", "accessList": []any{map[string]any{"address": "payer", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
		{"dynamic-list-auto", map[string]any{"do": "sendTx", "from": "payer", "to": "0x000000000000000000000000000000000000dead", "value": "5", "data": "0xabcdef", "nonce": "7", "maxFeePerGas": "3", "maxPriorityFeePerGas": "1", "accessList": []any{map[string]any{"address": "0x000000000000000000000000000000000000dead", "storageKeys": []any{"0x" + strings.Repeat("0", 63) + "1"}}}}, 1},
	} {
		t.Run(sample.name, func(t *testing.T) {
			t.Setenv("CHAINBENCH_TEST_ASSET_TOKEN", "asset-test-token")
			var signed, unsigned atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(out http.ResponseWriter, req *http.Request) {
				if req.Header.Get("Authorization") != "Bearer asset-test-token" {
					http.Error(out, "unauthorized", http.StatusUnauthorized)
					return
				}
				var request struct {
					ID     any               `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				var value any = "0x1"
				switch request.Method {
				case "eth_sendTransaction":
					unsigned.Add(1)
					_ = json.NewEncoder(out).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "error": map[string]any{"code": -32601, "message": "node signing is disabled"}})
					return
				case "eth_chainId":
					value = "0x205b"
				case "eth_getProof":
					value = map[string]string{"extra": "0x0"}
				case "eth_getTransactionCount":
					value = fmt.Sprintf("%#x", signed.Load())
				case "eth_estimateGas":
					if sample.name == "fee-create-refused" {
						_ = json.NewEncoder(out).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "error": map[string]any{"code": -32000, "message": "creation estimate refused"}})
						return
					}
					if strings.Contains(sample.name, "auto") && !strings.HasPrefix(sample.name, "deploy-") {
						var args map[string]any
						if err := json.Unmarshal(request.Params[0], &args); err != nil {
							t.Error(err)
						}
						for field, expected := range map[string]string{"nonce": "0x7", "gasPrice": "0x3", "maxFeePerGas": "0x3", "maxPriorityFeePerGas": "0x1"} {
							if _, supplied := sample.step[field]; supplied && args[field] != expected {
								t.Errorf("estimate lost %s: %v", field, args[field])
							}
						}
					}
					if sample.name == "fee-create-auto" {
						var args map[string]any
						_ = json.Unmarshal(request.Params[0], &args)
						if args["to"] != nil || args["type"] != "0x16" || args["feePayer"] == nil || args["data"] != sample.step["bytecode"] || args["value"] != "0x5" || args["nonce"] != "0x7" {
							t.Errorf("creation estimate lost intent or payer: %+v", args)
						}
					}
					value = "0x10000"
				case "eth_getBlockByNumber":
					value = map[string]string{"number": "0x1", "baseFeePerGas": "0x1", "gasLimit": "0xf4240", "gasUsed": "0x0"}
				case "eth_sendRawTransaction":
					if strings.HasPrefix(sample.name, "fee-") {
						private, _ := hex.DecodeString(strings.Repeat("0", 63) + "1")
						sender, _ := sdkcrypto.PrivKeyFromBytes(private)
						feePrivate, _ := hex.DecodeString(strings.Repeat("0", 63) + "2")
						sponsor, _ := sdkcrypto.PrivKeyFromBytes(feePrivate)
						to, _ := sdktypes.HexToAddress("0x000000000000000000000000000000000000dead")
						data, _ := hex.DecodeString("abcdef")
						gas := uint64(70000)
						if strings.HasSuffix(sample.name, "auto") || sample.name == "fee-create-default" {
							gas = 65536
						}
						target := &to
						if sample.step["do"] == "deployContract" {
							target = nil
							data, _ = hex.DecodeString("60006000f3")
						}
						list := sdktx.AccessList{{Address: to, StorageKeys: []sdktypes.Hash{sdktypes.BytesToHash([]byte{1})}}}
						nonce := uint64(7)
						if sample.name == "fee-create-default" {
							list = nil
							nonce = 0
						}
						expected := sdktx.FeeDelegateTx{Sender: sdktx.DynamicFeeTx{ChainID: big.NewInt(8283), Nonce: nonce, Gas: gas, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(3), Value: big.NewInt(5), Data: data, To: target, AccessList: list}}
						if err := expected.Sign(sender, sponsor); err != nil {
							t.Error(err)
						}
						encoded, err := expected.Encode()
						if err != nil {
							t.Error(err)
						}
						var actual string
						if err := json.Unmarshal(request.Params[0], &actual); err != nil {
							t.Error(err)
						}
						if actual != "0x"+hex.EncodeToString(encoded) {
							t.Error("fee-delegated signature lost sender/payer or explicit transaction fields")
						}
					}
					if strings.HasPrefix(sample.name, "dynamic-") {
						private, _ := hex.DecodeString(strings.Repeat("0", 63) + "1")
						key, _ := sdkcrypto.PrivKeyFromBytes(private)
						to, _ := sdktypes.HexToAddress("0x000000000000000000000000000000000000dead")
						data, _ := hex.DecodeString("abcdef")
						gas := uint64(70000)
						if sample.name == "dynamic-list-auto" || strings.HasPrefix(sample.name, "dynamic-auto-") {
							gas = 65536
						}
						listAddress := to
						if sample.name == "dynamic-list-label" {
							listAddress = sdktypes.BytesToAddress(sdkcrypto.PubkeyToAddress(key.PubKey()).Bytes())
						}
						target := &to
						if sample.name == "dynamic-create-list" {
							target = nil
							data, _ = hex.DecodeString("60006000f3")
						}
						expected := sdktx.DynamicFeeTx{ChainID: big.NewInt(8283), Nonce: 7, Gas: gas, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(3), Value: big.NewInt(5), Data: data, To: target, AccessList: sdktx.AccessList{{Address: listAddress, StorageKeys: []sdktypes.Hash{sdktypes.BytesToHash([]byte{1})}}}}
						if strings.HasPrefix(sample.name, "dynamic-auto-") {
							expected.AccessList = nil
						}
						if sample.name == "dynamic-auto-fees" {
							expected.Nonce = 0
						}
						if err := expected.Sign(key); err != nil {
							t.Error(err)
						}
						var actual string
						if err := json.Unmarshal(request.Params[0], &actual); err != nil {
							t.Error(err)
						}
						if actual != "0x"+hex.EncodeToString(expected.Encode()) {
							t.Error("dynamic access-list signature lost explicit fields or list contents")
						}
					}
					if strings.HasPrefix(sample.name, "deploy-auto-") {
						private, _ := hex.DecodeString(strings.Repeat("0", 63) + "1")
						key, _ := sdkcrypto.PrivKeyFromBytes(private)
						data, _ := hex.DecodeString("60006000f3")
						var encoded []byte
						if sample.name == "deploy-auto-legacy" {
							expected := sdktx.LegacyTx{Nonce: 7, Gas: 65536, GasPrice: big.NewInt(3), Value: big.NewInt(5), Data: data}
							if err := expected.Sign(big.NewInt(8283), key); err != nil {
								t.Error(err)
							}
							encoded = expected.Encode()
						} else {
							expected := sdktx.DynamicFeeTx{ChainID: big.NewInt(8283), Nonce: 7, Gas: 65536, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(3), Value: big.NewInt(5), Data: data}
							if err := expected.Sign(key); err != nil {
								t.Error(err)
							}
							encoded = expected.Encode()
						}
						var actual string
						if err := json.Unmarshal(request.Params[0], &actual); err != nil {
							t.Error(err)
						}
						if actual != "0x"+hex.EncodeToString(encoded) {
							t.Error("estimated deployment lost gas, nonce, fee model, value or code")
						}
					}
					if !strings.HasPrefix(sample.name, "deploy-auto-") && (sample.name == "legacy" || sample.name == "legacy-auto" || strings.HasSuffix(sample.name, "-legacy") || sample.name == "access-list") {
						private, _ := hex.DecodeString(strings.Repeat("0", 63) + "1")
						key, _ := sdkcrypto.PrivKeyFromBytes(private)
						to, _ := sdktypes.HexToAddress("0x000000000000000000000000000000000000dead")
						data, _ := hex.DecodeString("abcdef")
						var encoded []byte
						if sample.name != "access-list" {
							var target = &to
							gas, value := uint64(70000), big.NewInt(5)
							if sample.name == "legacy-auto" {
								gas = 65536
							}
							if sample.name == "faucet-legacy" {
								data = nil
							}
							if sample.name == "deploy-legacy" {
								target = nil
								data, _ = hex.DecodeString("60006000f3")
							}
							if sample.name == "load-legacy" {
								target = nil
								gas = 300000
								value = new(big.Int)
								data, _ = hex.DecodeString("60005b600101620186a05a1160025760006000f3")
							}
							expected := sdktx.LegacyTx{Nonce: uint64(7 + signed.Load()), Gas: gas, GasPrice: big.NewInt(3), Value: value, Data: data, To: target}
							if err := expected.Sign(big.NewInt(8283), key); err != nil {
								t.Error(err)
							}
							encoded = expected.Encode()
						} else {
							slot := sdktypes.BytesToHash([]byte{1})
							target := &to
							if sample.name == "access-list-create" {
								target = nil
								data, _ = hex.DecodeString("60006000f3")
							}
							expected := sdktx.AccessListTx{ChainID: big.NewInt(8283), Nonce: 7, Gas: 70000, GasPrice: big.NewInt(3), Value: big.NewInt(5), Data: data, To: target, AccessList: sdktx.AccessList{{Address: to, StorageKeys: []sdktypes.Hash{slot}}}}
							if err := expected.Sign(key); err != nil {
								t.Error(err)
							}
							encoded = expected.Encode()
						}
						var actual string
						if err := json.Unmarshal(request.Params[0], &actual); err != nil {
							t.Error(err)
						}
						if actual != "0x"+hex.EncodeToString(encoded) {
							t.Error("legacy/access-list signature lost data, gas, nonce, fees, or list contents")
						}
					}
					if sample.name == "load" {
						private, _ := hex.DecodeString(strings.Repeat("0", 63) + "1")
						key, _ := sdkcrypto.PrivKeyFromBytes(private)
						data, _ := hex.DecodeString("60005b600101620186a05a1160025760006000f3")
						expected := sdktx.DynamicFeeTx{ChainID: big.NewInt(8283), Nonce: uint64(7 + signed.Load()), Gas: 300000, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(3), Value: new(big.Int), Data: data}
						if err := expected.Sign(key); err != nil {
							t.Error(err)
						}
						var actual string
						if err := json.Unmarshal(request.Params[0], &actual); err != nil {
							t.Error(err)
						}
						if actual != "0x"+hex.EncodeToString(expected.Encode()) {
							t.Error("load signature lost gas/data/fees or consecutive explicit nonce")
						}
					}
					signed.Add(1)
					value = "0x" + strings.Repeat("ab", 32)
				case "eth_getTransactionReceipt":
					value = map[string]string{"status": "0x1", "contractAddress": "0x000000000000000000000000000000000000beef", "blockNumber": "0x1"}
				}
				_ = json.NewEncoder(out).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": value})
			}))
			defer server.Close()
			key := filepath.Join(t.TempDir(), "private")
			feeKey := filepath.Join(t.TempDir(), "sponsor-private")
			if err := os.WriteFile(feeKey, []byte(strings.Repeat("0", 63)+"2"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(key, []byte(strings.Repeat("0", 63)+"1"), 0600); err != nil {
				t.Fatal(err)
			}
			engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", ArtifactRoot: t.TempDir(), Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1, Role: node.RoleEN, RPCURL: server.URL, Auth: node.Auth{"type": "jwt", "env": "CHAINBENCH_TEST_ASSET_TOKEN"}}}}})
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "external-assets", "requires": []string{"rpc"}, "chainPreset": map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "external", "chain": "stablenet", "attach": map[string]any{"rpc": []string{server.URL}}, "accounts": map[string]any{"payer": map[string]any{"keyFile": key}, "sponsor": map[string]any{"keyFile": feeKey}}}, "steps": []map[string]any{sample.step, {"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "1"}}})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := dsl.Parse(raw); err != nil {
				t.Fatal("invalid fixture", err)
			}
			root, err := engine.Run(context.Background(), [][]byte{raw})
			if err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(filepath.Join(root, "tests", "001_external-assets", "status.json"))
			if err != nil {
				t.Fatal(err)
			}
			var verdict struct {
				Result string `json:"result"`
			}
			wanted := "pass"
			if sample.transactions == 0 {
				wanted = "fail"
			}
			if err := json.Unmarshal(content, &verdict); err != nil || verdict.Result != wanted || signed.Load() != sample.transactions || unsigned.Load() != 0 {
				t.Fatalf("declared key did not sign locally: %v %s signed=%d unsigned=%d", err, content, signed.Load(), unsigned.Load())
			}
		})
	}
}
