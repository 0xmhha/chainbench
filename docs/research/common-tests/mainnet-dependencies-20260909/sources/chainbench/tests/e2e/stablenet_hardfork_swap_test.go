//go:build e2e

package e2e

import (
	"math/big"
	"testing"
	"time"
)

// TestE2E_StablenetHardforkSwap is the stablenet BINARY-SWAP hardfork (former
// stablenet-hardfork-swap.sh, scenario 5): boot on the pre-fork gstable with the
// fork delayed to bohoBlock, deploy a contract, then swap every node to the
// post-fork build in place via `chainbench hardfork`. Afterwards production must
// cross the fork block, the pre-fork contract state must survive, and a post-fork
// tx must succeed.
//
// PRE_FORK_BIN (or GSTABLE_BIN) boots; POST_FORK_BIN is swapped in (default: the
// same binary — then this exercises the swap mechanics and state survival, with
// no fork-behavior delta). Distinct from delayed-fork, which uses one binary.
func TestE2E_StablenetHardforkSwap(t *testing.T) {
	pre := requireBinary(t, "GSTABLE_BIN", "gstable")
	if p := envOr("PRE_FORK_BIN", ""); p != "" {
		pre = p
	}
	post := envOr("POST_FORK_BIN", pre)
	cli := buildCLI(t)

	const boho int64 = 40
	n := boot(t, cli, "stablenet", pre, 4, 1, "genesis.overrides.bohoBlock=40")
	url := n.rpcURL
	key := presetFundedKey(t)

	// pre-fork production + a contract whose state must survive the swap.
	n.waitAdvancing(url, 45*time.Second)
	addr := n.deployReturns42(key, url)
	n.waitCallReturns42(url, addr, 90*time.Second)

	// the swap must happen before the fork block.
	if h := head(t, url); h >= boho {
		t.Fatalf("head (%d) already reached bohoBlock (%d) before swap — raise boho", h, boho)
	}

	// BINARY SWAP: stop pre-fork nodes, relaunch same datadirs on the post-fork
	// build in place (identity/config preserved).
	n.hardfork("stablenet", post, boho)
	url = n.rpcURLFor(1)

	// post-fork production continues and crosses the fork block.
	n.waitCross(url, boho, 120*time.Second)

	// pre-fork state survives the swap + fork.
	n.waitCallReturns42(url, addr, 30*time.Second)

	// post-fork tx processing.
	oneEth := new(big.Int).SetUint64(1_000_000_000_000_000_000)
	n.sendValue(key, url, deadAddr, oneEth)
}
