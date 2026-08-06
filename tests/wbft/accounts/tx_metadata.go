// This file ports three post-transfer RPC-metadata checks from the legacy bash
// regression suite (regression/g-api): the nonce increment
// (g1-05-get-tx-count), the eth_getTransactionByHash field population
// (g1-03-get-tx-by-hash), and the eth_getTransactionReceipt field population
// (g1-04-get-tx-receipt). Each drives one value transfer over the accounts SDK
// and then asserts on what the node reports about that transaction.
//
// # Test: transaction-count-increments
//
// Intent:   a mined value transfer advances the sender's nonce by exactly one.
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   read eth_getTransactionCount(sender, latest), send a 1-wei transfer,
//
//	wait for its receipt, then read the count again.
//
// Pass:     after - before == 1.
//
// # Test: transaction-by-hash-fields
//
// Intent:   after a transfer mines, eth_getTransactionByHash returns a fully
//
//	populated transaction object.
//
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   send a 1-wei transfer, wait for its receipt, then fetch the tx by
//
//	hash and assert blockNumber, from, to and value are all non-empty.
//
// Pass:     blockNumber, from, to and value are populated.
//
// # Test: transaction-receipt-fields
//
// Intent:   a transfer receipt reports the post-Byzantium status field plus
//
//	effectiveGasPrice and a logs array (the PR #70 receipt-shape regression).
//
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   send a 1-wei transfer, wait for its receipt, and assert the receipt
//
//	JSON carries status, effectiveGasPrice and a logs field.
//
// Pass:     status, effectiveGasPrice and logs are all present in the receipt.
//
// These are chainbench TEST CODE (requirement #16): they drive real
// transactions, so they are only meaningful against a live network (the sibling
// _test.go validates registration/gating, not the transactions themselves).
package accounts

import (
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// Distinct recipients so the three cases never contend on the same account.
const (
	txCountRecipient   = "0x00000000000000000000000000000000C0FFEE04"
	txByHashRecipient  = "0x00000000000000000000000000000000C0FFEE05"
	txReceiptRecipient = "0x00000000000000000000000000000000C0FFEE06"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "transaction-count-increments",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           transactionCountIncrements,
	})
	testkit.Register(testkit.Case{
		Name:         "transaction-by-hash-fields",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           transactionByHashFields,
	})
	testkit.Register(testkit.Case{
		Name:         "transaction-receipt-fields",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           transactionReceiptFields,
	})
}

// openFaucetWallet opens a wallet for the shared genesis-funded key against the
// primary node — the common prologue of the three metadata cases.
func openFaucetWallet(t *testkit.T) (accounts.Wallet, *rpc.Client) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")

	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")

	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")
	return w, rpc.Dial(primary.RPCURL)
}

// waitForReceipt blocks until the tx hash has a mined receipt or the case times
// out.
func waitForReceipt(t *testkit.T, c *rpc.Client, hash string) {
	t.WaitFor(func() bool {
		raw, err := c.TxReceipt(t.Ctx(), hash)
		return err == nil && raw != nil
	}, 90*time.Second, time.Second, "transaction receipt to be mined")
}

func transactionCountIncrements(t *testkit.T) {
	w, c := openFaucetWallet(t)

	before, err := c.NonceAt(t.Ctx(), w.Address())
	t.NoErr(err, "nonce before")

	hash, err := w.SendCoin(t.Ctx(), txCountRecipient, big.NewInt(1))
	t.NoErr(err, "value transfer")
	waitForReceipt(t, c, hash)

	after, err := c.NonceAt(t.Ctx(), w.Address())
	t.NoErr(err, "nonce after")

	t.Equalf(after-before, uint64(1), "sender nonce incremented by exactly 1")
}

func transactionByHashFields(t *testkit.T) {
	w, c := openFaucetWallet(t)

	hash, err := w.SendCoin(t.Ctx(), txByHashRecipient, big.NewInt(1))
	t.NoErr(err, "value transfer")
	waitForReceipt(t, c, hash)

	var tx struct {
		BlockNumber string `json:"blockNumber"`
		From        string `json:"from"`
		To          string `json:"to"`
		Value       string `json:"value"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getTransactionByHash", &tx, hash), "eth_getTransactionByHash")

	t.Truef(tx.BlockNumber != "", "transaction has blockNumber")
	t.Truef(tx.From != "", "transaction has from")
	t.Truef(tx.To != "", "transaction has to")
	t.Truef(tx.Value != "", "transaction has value")
}

func transactionReceiptFields(t *testkit.T) {
	w, c := openFaucetWallet(t)

	hash, err := w.SendCoin(t.Ctx(), txReceiptRecipient, big.NewInt(1))
	t.NoErr(err, "value transfer")
	waitForReceipt(t, c, hash)

	raw, err := c.TxReceipt(t.Ctx(), hash)
	t.NoErr(err, "eth_getTransactionReceipt")
	t.Truef(raw != nil, "receipt is present")

	var fields map[string]json.RawMessage
	t.NoErr(json.Unmarshal(raw, &fields), "decode receipt object")

	for _, name := range []string{"status", "effectiveGasPrice", "logs"} {
		v, ok := fields[name]
		t.Truef(ok && len(v) > 0 && strings.TrimSpace(string(v)) != "null",
			"receipt has non-null %s field", name)
	}
}
