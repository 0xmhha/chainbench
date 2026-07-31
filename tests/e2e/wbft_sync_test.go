//go:build e2e

package e2e

import (
	"testing"
	"time"
)

// TestE2E_WbftSnapSync ports wemix4 NODE-004 (snap sync) for the go-wbft binary:
// boot a wbft network with the endpoint in `snap` sync mode, stop it so the
// validators open a large block gap, restart it, and assert it re-syncs — head
// within 2 of a validator, matching hash/stateRoot at a block produced while it
// was down, readable state, and eth_syncing == false. A large gap (GAP env, ~64+)
// is what pushes geth into the snap-sync path rather than a short block replay.
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix GAP=90 go test -tags e2e -run TestE2E_WbftSnapSync -v ./tests/e2e/
func TestE2E_WbftSnapSync(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)
	gap := envInt("GAP", 90)

	n := boot(t, cli, "wbft", bin, 4, 1, "nodes.endpoint_syncmode=snap")
	bp := n.rpcURLFor(1) // a validator
	en := n.rpcURLFor(5) // the snap-sync endpoint
	n.waitAdvancing(bp, 60*time.Second)

	// Stop the endpoint to open a gap.
	n.nodeStop(5)
	stopHead := head(t, bp)

	// Let the validators advance at least GAP blocks past the stop.
	deadline := timeAfter(time.Duration(gap*3+60) * time.Second)
	for head(t, bp) < stopHead+gap {
		if deadline() {
			t.Fatalf("validators produced < %d blocks after endpoint stop", gap)
		}
		time.Sleep(2 * time.Second)
	}
	bpBefore := head(t, bp)

	// Restart the endpoint; wait until it catches up (head within 2).
	n.nodeStart(5)
	resynced := false
	d2 := timeAfter(180 * time.Second)
	for !d2() {
		enHead, bpHead := head(t, en), head(t, bp)
		diff := bpHead - enHead
		if diff < 0 {
			diff = -diff
		}
		if enHead >= 0 && diff <= 2 {
			resynced = true
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !resynced {
		t.Fatalf("snap endpoint did not re-sync within 180s (en=%d bp=%d)", head(t, en), head(t, bp))
	}

	// Hash + stateRoot agreement at a block produced while the endpoint was down.
	sample := bpBefore - 1
	if sample < 1 {
		sample = 1
	}
	sh := hexBlock(sample)
	if bpHash, enHash := blockField(bp, sh, "hash"), blockField(en, sh, "hash"); bpHash == "" || bpHash != enHash {
		t.Fatalf("block %d hash mismatch: bp=%s en=%s", sample, bpHash, enHash)
	}
	if bpRoot, enRoot := blockField(bp, sh, "stateRoot"), blockField(en, sh, "stateRoot"); bpRoot == "" || bpRoot != enRoot {
		t.Fatalf("block %d stateRoot mismatch: bp=%s en=%s", sample, bpRoot, enRoot)
	}

	// State access: a genesis-funded account (preset node 1) is readable on the
	// re-synced endpoint.
	if balance(t, en, "0xc17d493883eaa3b4cceb0f214b273392d562f9d8").Sign() == 0 {
		t.Fatalf("snap endpoint cannot read funded account state (balance 0)")
	}

	// Fully synced.
	if !synced(en) {
		t.Fatal("snap endpoint still reports eth_syncing != false")
	}
}
