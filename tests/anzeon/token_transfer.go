// This file adds a native-coin (WKRC) transfer + event case (ported from
// regression f-system-contracts f1-01/f1-04), exercising the full binding
// layer end to end: ABI call encoding, a contract Execute, and event-log
// decoding.
//
// # Test: token-transfer-emits-event
//
// Intent:   a native-coin adapter transfer(to, value) must move the tokens and
//
//	emit a Transfer(from, to, value) event decodable from the receipt.
//
// Applies:  stablenet (the native-coin adapter is the go-stablenet WKRC token).
// Requires: the "rpc" capability.
// Method:   Execute the adapter's transfer(address,uint256) for 1 wei to a fresh
//
//	recipient; read the receipt and assert a Transfer log whose indexed `to`
//	is the recipient and whose data value is 1.
//
// Pass:     the Transfer event is present and decodes to (to=recipient, value=1).
//
// This is chainbench TEST CODE (requirement #16): it drives a real transaction,
// so it is only meaningful against a live network (the sibling _test.go validates
// registration/gating).
package anzeon

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

// faucetKeyHex is a genesis-funded key in the stablenet preset alloc
// (keys/preset/metadata.json). TEST FIXTURE ONLY — it is the upstream
// go-ethereum test key and is public knowledge.
const faucetKeyHex = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"

// tokenRecipient starts unfunded so the transfer is unambiguous.
const tokenRecipient = "0x00000000000000000000000000000000C0FFEE04"

func init() {
	testkit.Register(testkit.Case{
		Name:         "token-transfer-emits-event",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           tokenTransferEmitsEvent,
	})
}

func tokenTransferEmitsEvent(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")

	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	// transfer(recipient, 1) on the native-coin adapter.
	data := accounts.EncodeCall("transfer(address,uint256)",
		accounts.AddressArg(tokenRecipient), big.NewInt(1).Bytes())
	callData, err := hex.DecodeString(strings.TrimPrefix(data, "0x"))
	t.NoErr(err, "decode calldata")

	execHash, err := w.Execute(t.Ctx(), nativeCoinAdapter, callData, nil)
	t.NoErr(err, "transfer execute")

	transferTopic := accounts.EventTopic("Transfer(address,address,uint256)")
	c := rpc.Dial(primary.RPCURL)
	t.WaitFor(func() bool {
		raw, err := c.TxReceipt(t.Ctx(), execHash)
		if err != nil || len(raw) == 0 {
			return false
		}
		var r struct {
			Status string         `json:"status"`
			Logs   []accounts.Log `json:"logs"`
		}
		if err := json.Unmarshal(raw, &r); err != nil || r.Status != "0x1" {
			return false
		}
		log, found := accounts.FindLog(r.Logs, transferTopic)
		if !found || len(log.Topics) < 3 {
			return false
		}
		// Transfer(from indexed, to indexed, value): topics[2] = to, data = value.
		if !strings.EqualFold(accounts.TopicToAddress(log.Topics[2]), tokenRecipient) {
			return false
		}
		v, ok := accounts.WordToBig(log.Data)
		return ok && v.Cmp(big.NewInt(1)) == 0
	}, 90*time.Second, time.Second, "Transfer event to recipient with value 1")
}
