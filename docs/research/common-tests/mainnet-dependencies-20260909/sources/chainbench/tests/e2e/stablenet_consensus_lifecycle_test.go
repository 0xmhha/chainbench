//go:build e2e

package e2e

import (
	"testing"
	"time"
)

// TestE2E_StablenetConsensusLifecycle is the WBFT liveness/lifecycle scenario
// (former stablenet-consensus-lifecycle.sh, regression b-08/09/10 + a1-04). With
// 4 validators WBFT tolerates f=1 and needs 2f+1=3 to commit:
//   - stop 1 validator (3 remain) → production continues (b-09) and the chain
//     stays linked (b-10); restart → resumes (a1-04);
//   - stop 2 validators (2 remain, < quorum) → production halts (b-08); restart
//     → recovers.
//
// Head checks poll (waitAdvancing) so a round-change transient after stopping a
// validator doesn't false-fail.
func TestE2E_StablenetConsensusLifecycle(t *testing.T) {
	bin := requireBinary(t, "GSTABLE_BIN", "gstable")
	cli := buildCLI(t)

	n := boot(t, cli, "stablenet", bin, 4, 1)
	obs := n.rpcURLFor(5) // an endpoint, so we can observe while validators are down

	// baseline
	n.waitAdvancing(obs, 45*time.Second)

	// b-09: stop one validator; the remaining 3 keep producing (poll past the
	// round-change that skips the stopped proposer).
	n.nodeStop(1)
	n.waitAdvancing(obs, 45*time.Second)

	// b-10: the chain stays linked across the round change (parentHash chain).
	tip := head(t, obs)
	for _, bn := range []int64{tip, tip - 1, tip - 2} {
		if bn < 1 {
			continue
		}
		child := blockField(obs, hexBlock(bn), "parentHash")
		parent := blockField(obs, hexBlock(bn-1), "hash")
		if parent == "" || child != parent {
			t.Fatalf("b-10: parentHash chain broken at block %d: child.parentHash=%s parent.hash=%s", bn, child, parent)
		}
	}

	// a1-04: restart node1; it rejoins and production continues.
	n.nodeStart(1)
	n.waitAdvancing(obs, 45*time.Second)

	// b-08: stop two validators (2 remain, < quorum 3) → production halts.
	n.nodeStop(1)
	n.nodeStop(2)
	if grewWithin(t, obs, 12*time.Second) {
		t.Fatal("b-08: production continued below quorum (should halt)")
	}

	// recovery: restart both → production resumes.
	n.nodeStart(1)
	n.nodeStart(2)
	n.waitAdvancing(obs, 60*time.Second)
}
