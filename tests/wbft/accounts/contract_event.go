// # Test: contract-event-emitted
//
// Intent:   a contract call that emits an event must surface that event in the
//
//	transaction receipt's logs, decodable by topic — the receipt-log path
//	behind the system-contract event regression (mint/burn/Transfer).
//
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   deploy a minimal contract whose runtime emits a single LOG1 with a
//
//	fixed topic on any call; wait for it to deploy; Execute it; then read the
//	transaction receipt and assert a log with that topic is present
//	(accounts.FindLog).
//
// Pass:     the execution receipt contains a log whose topic0 is the fixed topic.
//
// This is chainbench TEST CODE (requirement #16): it drives real transactions, so
// it is only meaningful against a live network (the sibling _test.go validates
// registration/gating).
package accounts

import (
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// eventTopic is the fixed 32-byte topic the test contract logs.
const eventTopic = "0x1111111111111111111111111111111111111111111111111111111111111111"

// eventContractInit deploys a runtime that emits LOG1(topic, 0, 0) then STOPs:
//
//	prefix  6027600c60003960276000f3   (return the 0x27-byte runtime)
//	runtime 7f<topic> 6000 6000 a1 00  (PUSH32 topic; size 0; offset 0; LOG1; STOP)
const eventContractInit = "6027600c60003960276000f3" +
	"7f1111111111111111111111111111111111111111111111111111111111111111" +
	"60006000a100"

func init() {
	testkit.Register(testkit.Case{
		Name:         "contract-event-emitted",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           contractEventEmitted,
	})
}

func contractEventEmitted(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")

	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	initCode, err := hex.DecodeString(eventContractInit)
	t.NoErr(err, "decode init code")
	_, addr, err := w.Deploy(t.Ctx(), initCode, nil)
	t.NoErr(err, "deploy event contract")

	c := rpc.Dial(primary.RPCURL)
	t.WaitFor(func() bool {
		code, err := c.CodeAt(t.Ctx(), addr)
		return err == nil && code != "" && code != "0x"
	}, 90*time.Second, time.Second, "event contract to deploy")

	execHash, err := w.Execute(t.Ctx(), addr, nil, nil)
	t.NoErr(err, "execute event contract")

	t.WaitFor(func() bool {
		raw, err := c.TxReceipt(t.Ctx(), execHash)
		if err != nil || len(raw) == 0 {
			return false
		}
		var r struct {
			Logs []accounts.Log `json:"logs"`
		}
		if err := json.Unmarshal(raw, &r); err != nil {
			return false
		}
		_, found := accounts.FindLog(r.Logs, eventTopic)
		return found
	}, 90*time.Second, time.Second, "execution receipt to contain the emitted event")
}
