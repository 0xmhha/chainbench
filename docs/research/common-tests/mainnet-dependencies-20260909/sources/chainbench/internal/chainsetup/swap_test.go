package chainsetup_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// TestNodeSwap_RelaunchesOnNewBinaryKeepingRevision: swapping one node onto a
// different binary stops it, relaunches from the same argv on the new binary,
// points the node's per-node binary at it, and keeps the pre-swap pid/command
// as a ledger revision.
func TestNodeSwap_RelaunchesOnNewBinaryKeepingRevision(t *testing.T) {
	dir, stub, deps := launchedNetwork(t)

	// Seed the ledger with the pre-swap launch so the swap supersedes it.
	pre, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatalf("open ledger: %v", err)
	}
	if err := pre.Record(process.Proc{
		PID: 1001, Label: string(node.LabelFor(1)), Binary: "gstable",
		Command: "/opt/gstable --datadir x", Host: "127.0.0.1",
	}); err != nil {
		t.Fatalf("seed ledger: %v", err)
	}
	if err := pre.Save(); err != nil {
		t.Fatalf("save seed ledger: %v", err)
	}

	const swapBin = "/opt/gwbft-swap"
	out, err := chainsetup.NodeSwap(context.Background(), deps, chainsetup.NodeSwapIn{
		DataDir: dir, Index: 1, Binary: swapBin,
	})
	if err != nil {
		t.Fatalf("NodeSwap: %v", err)
	}
	if out.Node.Index != 1 || out.Node.PID != 2001 {
		t.Errorf("swapped node = %+v, want index 1 pid 2001", out.Node)
	}
	// The driver stopped the old process and launched the new one.
	if len(stub.stopped) != 1 || stub.stopped[0] != 1 {
		t.Errorf("driver stopped %v, want [1]", stub.stopped)
	}

	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatalf("reopen workspace: %v", err)
	}
	st := ws.State()
	// node1 now points at the swapped binary; node2 is untouched.
	n1 := st.Nodes[0]
	if n1.Index != 1 {
		t.Fatalf("node order changed: %+v", st.Nodes)
	}
	if got := st.Binaries[n1.Binary]; got != swapBin {
		t.Errorf("node1 binary = %q (key %q), want %q", got, n1.Binary, swapBin)
	}

	// The ledger kept the pre-swap launch as a revision.
	led, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatalf("reopen ledger: %v", err)
	}
	label := string(node.LabelFor(1))
	cur, ok := led.Get(label)
	if !ok || cur.PID != 2001 || cur.Revision != 1 {
		t.Fatalf("current entry = %+v ok=%v, want pid 2001 rev 1", cur, ok)
	}
	if cur.Binary != "gwbft-swap" {
		t.Errorf("current binary = %q, want gwbft-swap", cur.Binary)
	}
	hist := led.History(label)
	if len(hist) != 1 || hist[0].PID != 1001 {
		t.Fatalf("history = %+v, want one pre-swap entry with pid 1001", hist)
	}
}

// TestNodeSwap_RequiresBinary: a swap with no binary is an error, not a silent
// relaunch on the same one.
func TestNodeSwap_RequiresBinary(t *testing.T) {
	dir, _, deps := launchedNetwork(t)
	if _, err := chainsetup.NodeSwap(context.Background(), deps, chainsetup.NodeSwapIn{DataDir: dir, Index: 1}); err == nil {
		t.Fatal("want an error swapping with no binary")
	}
}
