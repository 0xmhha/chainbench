package testhelper

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func sendDelegatedOptions(ctx context.Context, ac *interp.ActionCtx, wallet accounts.Wallet, to string, value *big.Int, data, url string) (string, error) {
	var key []byte
	if label, ok := ac.Args["feePayer"].(string); ok && label != "" {
		payer, err := ResolveAccount(ac.Deps, label)
		if err != nil {
			return "", err
		}
		if !payer.SignsLocally() {
			return "", fmt.Errorf("dsl: feePayer needs a declared local key")
		}
		key = payer.Key
	} else {
		literal, _ := ac.Args["feePayerKey"].(string)
		decoded, err := hex.DecodeString(strings.TrimPrefix(literal, "0x"))
		if err != nil {
			return "", fmt.Errorf("dsl: feePayerKey: %w", err)
		}
		key = decoded
	}
	if value == nil {
		value = new(big.Int)
	}
	custom := to == "" // Contract creation must use the full envelope, including its init code.
	for _, field := range []string{"gas", "data", "nonce", "maxFeePerGas", "maxPriorityFeePerGas", "accessList"} {
		if _, supplied := ac.Args[field]; supplied {
			custom = true
		}
	}
	if !custom {
		return wallet.SendFeeDelegated(ctx, key, to, value)
	}
	var list []accounts.AccessTuple
	if raw, supplied := ac.Args["accessList"]; supplied {
		decoded, err := localAccessList(ac.Deps, raw)
		if err != nil {
			return "", err
		}
		list = decoded
	}
	gas, explicit := hexQuantity(ac.Args["gas"])
	if !explicit {
		message := rpc.SendTxArgs{From: wallet.Address(), To: to, Value: "0x" + value.Text(16), Data: data, AccessList: list}
		if err := applyFeeArgs(&message, ac.Args); err != nil {
			return "", err
		}
		encoded, err := json.Marshal(message)
		if err != nil {
			return "", err
		}
		var request map[string]any
		if err := json.Unmarshal(encoded, &request); err != nil {
			return "", err
		}
		payer, err := ac.Deps.Accounts.AddressForKey(key)
		if err != nil {
			return "", err
		}
		request["feePayer"], request["type"] = payer, "0x16"
		client, err := clientFor(ac.Deps, url)
		if err != nil {
			return "", err
		}
		if err := client.Call(ctx, "eth_estimateGas", &gas, request); err != nil {
			return "", err
		}
	}
	return sendSpecifiedGas(ctx, ac, wallet, to, value, data, gas, url, list, key)
}
