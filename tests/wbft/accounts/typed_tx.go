// This file ports the EIP-1559 dynamic-fee (0x2) and EIP-2930 access-list (0x1)
// transaction cases (from regression a-ethereum a2-02, a2-03), over the
// accounts SDK tx-type methods.
//
// # Test: dynamic-fee-tx
//
// Intent:   a type 0x02 (EIP-1559) value transfer is accepted and mines, and the
//
//	node reports its type as 0x2.
//
// Applies:  stablenet, wbft (the wbft family). Requires: the "rpc" capability.
// Method:   OpenWallet(faucetKey).SendDynamicFee(recipient, 1); wait for the tx
//
//	and assert eth_getTransactionByHash reports type 0x2.
//
// Pass:     the transaction is found with type 0x2.
//
// # Test: access-list-tx
//
// Intent:   a type 0x01 (EIP-2930) value transfer is accepted and mines, and the
//
//	node reports its type as 0x1.
//
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   OpenWallet(faucetKey).SendAccessList(recipient, 1); assert
//
//	eth_getTransactionByHash reports type 0x1.
//
// Pass:     the transaction is found with type 0x1.
//
// These are chainbench TEST CODE (requirement #16): they drive real transactions,
// so they are only meaningful against a live network (the sibling _test.go
// validates registration/gating).
package accounts

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/testkit"
)

const (
	dynFeeRecipient     = "0x00000000000000000000000000000000C0FFEE0C"
	accessListRecipient = "0x00000000000000000000000000000000C0FFEE0D"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "dynamic-fee-tx",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           func(t *testkit.T) { typedTxHasType(t, "SendDynamicFee", dynFeeRecipient, "0x2") },
	})
	testkit.Register(testkit.Case{
		Name:         "access-list-tx",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           func(t *testkit.T) { typedTxHasType(t, "SendAccessList", accessListRecipient, "0x1") },
	})
}

// typedTxHasType opens a faucet wallet, sends a 1-wei transfer with the named
// tx-type method, and asserts the mined transaction reports wantType.
func typedTxHasType(t *testkit.T, method, recipient, wantType string) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	var hash string
	switch method {
	case "SendDynamicFee":
		hash, err = w.SendDynamicFee(t.Ctx(), recipient, big.NewInt(1))
	case "SendAccessList":
		hash, err = w.SendAccessList(t.Ctx(), recipient, big.NewInt(1))
	}
	t.NoErr(err, method)

	t.WaitFor(func() bool {
		var raw json.RawMessage
		if err := t.Primary().Call(t.Ctx(), "eth_getTransactionByHash", &raw, hash); err != nil {
			return false
		}
		if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
			return false
		}
		var tx struct {
			Type string `json:"type"`
		}
		return json.Unmarshal(raw, &tx) == nil && tx.Type == wantType
	}, 90*time.Second, time.Second, "transaction mined with type "+wantType)
}
