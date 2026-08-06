//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// TestE2E_WbftAdminPeers ports wemix4 RPC-019 (admin_peers): on a multi-node
// network each node peers with the others, so admin_peers reports a non-empty
// peer set. It boots 4 nodes and polls node 1's admin_peers until it reports its
// peers (peering takes a moment after launch).
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftAdminPeers -v ./tests/e2e/
func TestE2E_WbftAdminPeers(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)

	n := boot(t, cli, "wbft", bin, 3, 1) // 4 nodes total
	url := n.rpcURL

	n.waitAdvancing(url, 60*time.Second)

	c := rpc.Dial(url)
	deadline := time.Now().Add(60 * time.Second)
	var peers int
	for time.Now().Before(deadline) {
		if peers = adminPeerCount(c); peers > 0 {
			return
		}
		time.Sleep(3 * time.Second)
	}
	t.Fatalf("admin_peers reported no peers on a 4-node network (last=%d)", peers)
}

// TestE2E_WbftIsValidator ports wemix4 RPC-023 (istanbul_isValidator): a node in
// the active validator set reports true, while a non-validator endpoint node
// reports false. It boots 4 validators + 1 endpoint and checks a validator's RPC
// (node 1) against the endpoint's (node 5).
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftIsValidator -v ./tests/e2e/
func TestE2E_WbftIsValidator(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)

	n := boot(t, cli, "wbft", bin, 4, 1) // 4 validators + 1 endpoint
	bp := n.rpcURLFor(1)                 // a validator
	en := n.rpcURLFor(5)                 // the endpoint (non-validator)
	n.waitAdvancing(bp, 60*time.Second)

	if !isValidator(t, bp) {
		t.Fatal("validator node reported istanbul_isValidator=false, want true")
	}
	if isValidator(t, en) {
		t.Fatal("endpoint node reported istanbul_isValidator=true, want false")
	}
}

// isValidator calls istanbul_isValidator on a node's RPC.
func isValidator(t *testing.T, url string) bool {
	t.Helper()
	var out bool
	if err := rpc.Dial(url).Call(context.Background(), "istanbul_isValidator", &out); err != nil {
		t.Fatalf("istanbul_isValidator(%s): %v", url, err)
	}
	return out
}

// adminPeerCount returns the number of peers admin_peers reports (0 on error).
func adminPeerCount(c *rpc.Client) int {
	var peers []struct {
		ID string `json:"id"`
	}
	if err := c.Call(context.Background(), "admin_peers", &peers); err != nil {
		return 0
	}
	return len(peers)
}
