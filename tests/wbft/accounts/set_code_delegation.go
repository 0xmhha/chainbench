// # Test: set-code-delegation
//
// Intent:   exercise the type 0x04 (EIP-7702 set-code) transaction path: a fresh
//
//	authority delegates its account code to a fixed address, sponsored by a
//	funded wallet. Ported from regression/a-ethereum (a2 set-code) and
//	the accounts SDK's 7702 check.
//
// Applies:  stablenet, wbft (all three target chains accept 0x00-0x04 + 0x16).
// Requires: the "rpc" capability.
// Method:   generate a fresh authority, SendSetCode delegating it to a fixed
//
//	address; assert the tx reports type 0x4, then assert the authority's code
//	became the 7702 delegation indicator 0xef0100||delegate.
//
// Pass:     tx type == 0x4 and authority code == 0xef0100||delegate.
//
// This is chainbench TEST CODE (requirement #16): it drives a real transaction,
// so it is only meaningful against a live network (the sibling _test.go
// validates registration/gating).
package accounts

import (
	"encoding/hex"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

const setCodeDelegate = "0x1111111111111111111111111111111111111111"

func init() {
	testkit.Register(testkit.Case{
		Name:         "set-code-delegation",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           setCodeDelegation,
	})
}

func setCodeDelegation(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	t.Truef(ap.SupportsTxType(0x04), "chain %s must support set-code (0x04)", t.NodeSet().Chain)

	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	// A fresh authority (nonce 0, unfunded — the sponsor pays gas).
	authKey, authAddr, err := accounts.GenerateKey()
	t.NoErr(err, "generate authority key")

	txHash, err := w.SendSetCode(t.Ctx(), authKey, setCodeDelegate)
	t.NoErr(err, "set-code transaction")

	var tx struct {
		Type string `json:"type"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getTransactionByHash", &tx, txHash), "eth_getTransactionByHash")
	t.Equalf(tx.Type, "0x4", "transaction type is set-code (0x4)")

	// After a valid authorization the authority's code becomes 0xef0100||delegate.
	wantCode := "0xef0100" + strings.ToLower(strings.TrimPrefix(setCodeDelegate, "0x"))
	t.WaitFor(func() bool {
		var code string
		if err := t.Primary().Call(t.Ctx(), "eth_getCode", &code, authAddr, "latest"); err != nil {
			return false
		}
		return strings.EqualFold(code, wantCode)
	}, 90*time.Second, time.Second, "authority code to become the 7702 delegation indicator")
}
