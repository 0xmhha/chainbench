//go:build e2e

package e2e

import (
	"testing"
	"time"
)

// TestE2E_WbftFaultTolerance ports wemix4 WBFT-007 (fault under 1/3): on a
// minimal BFT network (4 validators, quorum = ceil(2*4/3) = 3, fault tolerance
// f = 1), stopping a single validator leaves 3/4 online — exactly quorum — so
// consensus MUST continue. The stopped node then rejoins and re-syncs.
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftFaultTolerance -v ./tests/e2e/
func TestE2E_WbftFaultTolerance(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)

	n := boot(t, cli, "wbft", bin, 4, 1)
	url := n.rpcURL // validator 1 — stays up throughout

	// Warm up: the network is producing.
	n.waitAdvancing(url, 60*time.Second)

	// Stop validator 4: 3/4 remain == quorum, so consensus continues.
	n.nodeStop(4)
	n.waitAdvancing(url, 60*time.Second)

	// Restart it and confirm it catches back up to the live head.
	n.nodeStart(4)
	target := head(t, url)
	n.waitCross(n.rpcURLFor(4), target, 90*time.Second)
}

// TestE2E_WbftFaultHalt ports wemix4 WBFT-008 (fault over 1/3): on the same
// 4-validator network, stopping 2 validators leaves 2/4 online — below the
// quorum of 3 — so block production MUST halt. Restarting them resumes consensus.
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftFaultHalt -v ./tests/e2e/
func TestE2E_WbftFaultHalt(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)

	n := boot(t, cli, "wbft", bin, 4, 1)
	url := n.rpcURL // validator 1 — the observer

	n.waitAdvancing(url, 60*time.Second)

	// Stop validators 3 and 4: only 2/4 remain, below the quorum of 3.
	n.nodeStop(3)
	n.nodeStop(4)

	// Production must halt: the head does not grow over a generous window. Allow a
	// brief settle for any in-flight block to land before sampling.
	time.Sleep(5 * time.Second)
	if grewWithin(t, url, 20*time.Second) {
		t.Fatalf("consensus kept producing with 2/4 validators down (quorum 3 not met)")
	}

	// Restart both: quorum is restored and consensus resumes.
	n.nodeStart(3)
	n.nodeStart(4)
	n.waitAdvancing(url, 90*time.Second)
}
