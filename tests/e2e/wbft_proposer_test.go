//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestE2E_WbftRoundRobinProposer ports wemix4 WBFT-006 (round-robin proposer):
// over a window of consecutive blocks on a 4-validator network the block
// proposer (miner) must rotate — a healthy WBFT round-robin visits every
// validator, so several distinct proposers appear rather than one node sealing
// every block.
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftRoundRobinProposer -v ./tests/e2e/
func TestE2E_WbftRoundRobinProposer(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)

	n := boot(t, cli, "wbft", bin, 4, 1)
	url := n.rpcURL

	n.waitAdvancing(url, 60*time.Second)
	start := head(t, url)
	if start < 1 {
		start = 1
	}

	const window = 16
	n.waitCross(url, start+window, 120*time.Second)

	seen := map[string]int{}
	for b := start; b < start+window; b++ {
		m := strings.ToLower(blockField(url, hexBlock(b), "miner"))
		if m != "" {
			seen[m]++
		}
	}
	// With 4 validators over 16 blocks a round-robin proposer should visit at
	// least 3 of them (allowing for a warm-up round change); a single sealer is a
	// clear failure.
	if len(seen) < 3 {
		t.Fatalf("round-robin: expected >= 3 distinct proposers over %d blocks, saw %d: %v", window, len(seen), seen)
	}
}

// TestE2E_WbftViewChange ports wemix4 WBFT-003 (view change): killing the current
// proposer must trigger a round change that elects a new proposer, so the chain
// keeps advancing (observed from a surviving validator) and the blocks produced
// after the change stay linked by parentHash. Restarting the node restores it.
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftViewChange -v ./tests/e2e/
func TestE2E_WbftViewChange(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)

	n := boot(t, cli, "wbft", bin, 4, 1)
	obs := n.rpcURLFor(2) // observe from validator 2 (stays up)

	n.waitAdvancing(obs, 60*time.Second)
	before := head(t, obs)

	// Kill validator 1 (a proposer); 3/4 remain == quorum, so a view change must
	// keep consensus going.
	n.nodeStop(1)
	n.waitCross(obs, before+3, 90*time.Second)

	// The blocks produced across the round change stay chained by parentHash.
	cur := head(t, obs)
	for b := cur - 3; b < cur; b++ {
		parent := blockField(obs, hexBlock(b+1), "parentHash")
		prevHash := blockField(obs, hexBlock(b), "hash")
		if parent == "" || parent != prevHash {
			t.Fatalf("view change broke the chain: block %d parentHash %q != block %d hash %q", b+1, parent, b, prevHash)
		}
	}

	// Restart the killed proposer.
	n.nodeStart(1)
}
