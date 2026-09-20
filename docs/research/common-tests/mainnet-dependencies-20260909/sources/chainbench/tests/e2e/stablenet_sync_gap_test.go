//go:build e2e

package e2e

import (
	"testing"
	"time"
)

// allocAccount is a genesis-funded account no case spends — used to confirm the
// re-synced endpoint can read state.
const allocAccount = "0x71562b71999873db5b286df957af199ec94617f7"

// TestE2E_StablenetSyncGap is the endpoint re-sync scenario (former
// stablenet-sync-gap.sh, regression a1-02 full / a1-06 downloader / a1-03 snap):
// stop an endpoint, let the validators open a >= GAP block gap, restart it, and
// assert it re-syncs — head within 2 of a validator, matching block hash and
// stateRoot at a block produced while it was down, readable state, and
// eth_syncing == false.
//
//	# full sync (default); snap: SYNCMODE=snap GAP=150
//	GSTABLE_BIN=/path/to/gstable go test -tags e2e -run TestE2E_StablenetSyncGap -v ./tests/e2e/
func TestE2E_StablenetSyncGap(t *testing.T) {
	bin := requireBinary(t, "GSTABLE_BIN", "gstable")
	cli := buildCLI(t)
	syncmode := envOr("SYNCMODE", "full")
	gap := envInt("GAP", 12)

	n := boot(t, cli, "stablenet", bin, 4, 1, "nodes.endpoint_syncmode="+syncmode)
	bp := n.rpcURLFor(1) // a validator
	en := n.rpcURLFor(5) // the endpoint
	n.waitAdvancing(bp, 45*time.Second)

	// Stop the endpoint to open a gap.
	n.nodeStop(5)
	stopHead := head(t, bp)

	// Wait for the validators to advance at least GAP blocks past the stop.
	deadline := timeAfter(time.Duration(gap*3+40) * time.Second)
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
	d2 := timeAfter(120 * time.Second)
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
		time.Sleep(time.Second)
	}
	if !resynced {
		t.Fatalf("endpoint did not re-sync within 120s (en=%d bp=%d)", head(t, en), head(t, bp))
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

	// State access: a genesis-funded account is readable on the re-synced endpoint.
	if balance(t, en, allocAccount).Sign() == 0 {
		t.Fatalf("endpoint cannot read alloc account state (balance 0)")
	}

	// Fully synced.
	if !synced(en) {
		t.Fatal("endpoint still reports eth_syncing != false")
	}
}
