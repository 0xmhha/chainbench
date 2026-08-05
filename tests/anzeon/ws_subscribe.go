// This file ports the WebSocket subscription cases (from regression
// a-ethereum a4-06, a4-07). They open a WebSocket to the node, eth_subscribe to
// newHeads / logs, and assert notifications stream in. The WS endpoint is derived
// from the primary node's WS port, so the cases gate on the "ws" capability
// (advertised for launched networks; absent for pure-attach networks).
//
// # Test: ws-subscribe-new-heads (a4-06)
//
// Intent:   an eth_subscribe("newHeads") streams block headers.
// Applies:  stablenet. Requires "rpc" and "ws".
// Method:   subscribe over the node's WebSocket and assert a header notification
//
//	with a block number arrives.
//
// # Test: ws-subscribe-logs (a4-07)
//
// Intent:   an eth_subscribe("logs") streams matching logs.
// Applies:  stablenet. Requires "rpc" and "ws".
// Method:   subscribe to NativeCoinAdapter logs, emit a Transfer (a faucet token
//
//	transfer), and assert a log notification arrives.
//
// These are chainbench TEST CODE (requirement #16): live WebSocket flows, so the
// sibling _test.go validates registration/gating.
package anzeon

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func init() {
	reg := func(name string, fn func(*testkit.T)) {
		testkit.Register(testkit.Case{
			Name:         name,
			Category:     "api",
			ChainCompat:  []string{"stablenet"},
			RequiresCaps: []string{"rpc", "ws"},
			Fn:           fn,
		})
	}
	reg("ws-subscribe-new-heads", wsSubscribeNewHeads)
	reg("ws-subscribe-logs", wsSubscribeLogs)
}

// wsURL builds the primary node's WebSocket endpoint from its host and WS port.
func wsURL(t *testkit.T) string {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	t.Truef(primary.Ports.WS > 0, "primary node has no WS port")
	return fmt.Sprintf("ws://%s:%d", primary.Host, primary.Ports.WS)
}

func wsSubscribeNewHeads(t *testkit.T) {
	sub, err := rpc.Subscribe(t.Ctx(), wsURL(t), "newHeads")
	t.NoErr(err, "eth_subscribe newHeads")
	defer sub.Close()

	select {
	case n := <-sub.Notifications():
		var head struct {
			Number string `json:"number"`
		}
		t.NoErr(json.Unmarshal(n, &head), "decode newHeads notification")
		t.Truef(head.Number != "" && head.Number != "0x0",
			"newHeads notification carries a block number (got %q)", head.Number)
	case <-time.After(30 * time.Second):
		t.Fatalf("no newHeads notification within 30s")
	}
}

func wsSubscribeLogs(t *testkit.T) {
	// Subscribe first so the emitted log is captured.
	sub, err := rpc.Subscribe(t.Ctx(), wsURL(t), "logs", map[string]any{"address": nativeCoinAdapter})
	t.NoErr(err, "eth_subscribe logs")
	defer sub.Close()

	// Emit a Transfer by moving a little native coin through the adapter.
	w := openFaucetWallet(t)
	data, err := hex.DecodeString(strings.TrimPrefix(
		accounts.EncodeCallArgs("transfer(address,uint256)", accounts.Address(gastipRecipient), accounts.Uint(big.NewInt(1))), "0x"))
	t.NoErr(err, "encode transfer calldata")
	_, err = w.Execute(t.Ctx(), nativeCoinAdapter, data, nil)
	t.NoErr(err, "emit Transfer via NativeCoinAdapter")

	select {
	case n := <-sub.Notifications():
		var lg struct {
			Address string `json:"address"`
		}
		t.NoErr(json.Unmarshal(n, &lg), "decode log notification")
		t.Truef(strings.EqualFold(lg.Address, nativeCoinAdapter),
			"log notification is from the NativeCoinAdapter (got %s)", lg.Address)
	case <-time.After(30 * time.Second):
		t.Fatalf("no log notification within 30s")
	}
}
