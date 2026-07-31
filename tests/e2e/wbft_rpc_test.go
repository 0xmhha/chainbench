//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/pkg/core/rpc"
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
