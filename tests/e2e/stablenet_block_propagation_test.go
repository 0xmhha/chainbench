//go:build e2e

package e2e

import (
	"testing"
	"time"
)

// TestE2E_StablenetBlockPropagation is the near-head block-propagation check
// (former stablenet-block-propagation.sh, regression a1-07): over several rounds,
// every node's head stays within LAG of node 1's head — i.e. NewBlock /
// NewBlockHashes propagate to the whole set in real time.
func TestE2E_StablenetBlockPropagation(t *testing.T) {
	bin := requireBinary(t, "GSTABLE_BIN", "gstable")
	cli := buildCLI(t)

	const (
		validators = 4
		endpoints  = 1
		rounds     = 5
		lag        = 2 // max blocks a node may trail node 1
	)
	n := boot(t, cli, "stablenet", bin, validators, endpoints)

	urls := make([]string, 0, validators+endpoints)
	for i := 1; i <= validators+endpoints; i++ {
		urls = append(urls, n.rpcURLFor(i))
	}
	// Every node, not just node 1. What follows measures how far behind a node
	// runs once the network is up, and a node that has not started following
	// yet is not behind - it is absent. Waiting on node 1 alone let the first
	// sample land while the endpoint was still joining, which read as a
	// propagation failure at round 0.
	for _, u := range urls {
		n.waitAdvancing(u, 60*time.Second)
	}

	for r := 0; r < rounds; r++ {
		h1 := head(t, urls[0])
		for i, u := range urls {
			if i == 0 {
				continue
			}
			if h := head(t, u); h1-h > lag {
				t.Fatalf("round %d: node %d head %d trails node 1 head %d by more than %d",
					r, i+1, h, h1, lag)
			}
		}
		time.Sleep(2 * time.Second)
	}
}
