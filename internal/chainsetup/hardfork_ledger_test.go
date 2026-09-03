package chainsetup_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// TestHardforkExecute_SupersedesTheLedgerKeepingThePriorRevision: a hardfork
// swaps the binary in place, so the ledger must keep what ran before the fork
// as a revision rather than overwrite it. The pre-fork pid and binary land in
// history; the current entry is the post-fork one at revision 1.
func TestHardforkExecute_SupersedesTheLedgerKeepingThePriorRevision(t *testing.T) {
	dir, _, deps := launchedNetwork(t)

	// Seed the run ledger with the pre-fork launch the way a real start would
	// leave it (the test's seedWorkspace only fills the state view).
	pre, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatalf("open ledger: %v", err)
	}
	for _, n := range []struct {
		index, pid int
	}{{1, 1001}, {2, 1002}} {
		if err := pre.Record(process.Proc{
			PID: n.pid, Label: string(node.LabelFor(n.index)), Binary: "gstable",
			Command: "/opt/gstable --datadir x", Host: "127.0.0.1",
		}); err != nil {
			t.Fatalf("seed ledger node%d: %v", n.index, err)
		}
	}
	if err := pre.Save(); err != nil {
		t.Fatalf("save seed ledger: %v", err)
	}

	planned, err := chainsetup.HardforkPlan(context.Background(), deps, chainsetup.HardforkPlanIn{
		DataDir: dir, ToChain: "wbft", Block: 100,
	})
	if err != nil {
		t.Fatalf("HardforkPlan: %v", err)
	}
	const postFork = "/opt/gwbft"
	if _, err := chainsetup.HardforkExecute(context.Background(), deps, chainsetup.HardforkExecuteIn{
		Plan: planned, DataDir: dir, Binary: postFork,
	}); err != nil {
		t.Fatalf("HardforkExecute: %v", err)
	}

	after, err := process.OpenLedger(dir)
	if err != nil {
		t.Fatalf("reopen ledger: %v", err)
	}
	for _, index := range []int{1, 2} {
		label := string(node.LabelFor(index))
		cur, ok := after.Get(label)
		if !ok {
			t.Fatalf("node%d has no current ledger entry", index)
		}
		if cur.PID != 2000+index {
			t.Errorf("node%d current pid = %d, want post-fork %d", index, cur.PID, 2000+index)
		}
		if cur.Binary != "gwbft" {
			t.Errorf("node%d current binary = %q, want gwbft", index, cur.Binary)
		}
		if cur.Revision != 1 {
			t.Errorf("node%d current revision = %d, want 1", index, cur.Revision)
		}
		hist := after.History(label)
		if len(hist) != 1 {
			t.Fatalf("node%d history = %d entries, want 1 (the pre-fork launch)", index, len(hist))
		}
		if hist[0].PID != 1000+index || hist[0].Binary != "gstable" {
			t.Errorf("node%d history[0] = pid %d binary %q, want pid %d / gstable", index, hist[0].PID, hist[0].Binary, 1000+index)
		}
	}
}
