package chainsetup

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// wsForReuse builds a workspace whose recorded state matches the given snapshot
// baseline, so reconcileReuse compares like against like. Every node runs the
// same binary path, so binaryFor resolves to it. It carries a real (empty)
// ledger, since reconcile clears a redone node from the ledger.
func wsForReuse(t *testing.T, nodes []node.Record, genesisPath, genesisHash string, configHash map[int]string) *Workspace {
	t.Helper()
	li := map[string]string{genesisPath: genesisHash}
	for _, n := range nodes {
		li[n.ConfigPath] = configHash[n.Index]
	}
	l, err := process.OpenLedger(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := &Workspace{state: State{
		Binary:       "/bin/gwbft",
		GenesisPath:  genesisPath,
		Nodes:        nodes,
		LaunchInputs: li,
	}}
	w.ledger = l
	return w
}

func TestReconcileReuse_AllMatchLeavesRunningNodesUntouched(t *testing.T) {
	nodes := []node.Record{
		{Index: 1, Label: "bp1", ConfigPath: "/c/1.toml", PID: 111},
		{Index: 2, Label: "bp2", ConfigPath: "/c/2.toml", PID: 222},
	}
	cfg := map[int]string{1: "h1", 2: "h2"}
	w := wsForReuse(t, nodes, "/g/genesis.json", "gh", cfg)
	snap := reuseSnapshot{
		genesisHash: "gh",
		before: map[int]nodeBaseline{
			1: {Index: 1, ConfigHash: "h1", Binary: "/bin/gwbft", PID: 111},
			2: {Index: 2, ConfigHash: "h2", Binary: "/bin/gwbft", PID: 222},
		},
		alive: map[int]bool{1: true, 2: true},
	}
	plan, err := w.reconcileReuse(context.Background(), snap)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if len(plan.redo()) != 0 {
		t.Fatalf("redo = %v, want none", plan.redo())
	}
	// A reused node is left running: its recorded pid is untouched.
	if w.state.Nodes[0].PID != 111 || w.state.Nodes[1].PID != 222 {
		t.Fatalf("reused pids changed: %d, %d", w.state.Nodes[0].PID, w.state.Nodes[1].PID)
	}
}

func TestReconcileReuse_GenesisChangeRefusesAndTouchesNothing(t *testing.T) {
	nodes := []node.Record{{Index: 1, Label: "bp1", ConfigPath: "/c/1.toml", PID: 999}}
	w := wsForReuse(t, nodes, "/g/genesis.json", "gh-new", map[int]string{1: "h1"})
	snap := reuseSnapshot{
		genesisHash: "gh-old",
		before:      map[int]nodeBaseline{1: {Index: 1, ConfigHash: "h1", Binary: "/bin/gwbft", PID: 111}},
		alive:       map[int]bool{1: true},
	}
	plan, err := w.reconcileReuse(context.Background(), snap)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if plan.Refuse == "" {
		t.Fatal("expected refuse on genesis change")
	}
	// A refusal must not rewrite the recorded pid.
	if w.state.Nodes[0].PID != 999 {
		t.Fatalf("refusal touched node pid: %d", w.state.Nodes[0].PID)
	}
}

// TestReconcileReuse_StoppedDriftedNodeClearedFromLedger: a node whose config
// drifted but was not running (pid 0) is a redo — dropped from the ledger and
// left at pid 0, needing no process stop — while the matching node is left
// running.
func TestReconcileReuse_StoppedDriftedNodeClearedFromLedger(t *testing.T) {
	nodes := []node.Record{
		{Index: 1, Label: "bp1", ConfigPath: "/c/1.toml", PID: 111},
		{Index: 2, Label: "bp2", ConfigPath: "/c/2.toml", PID: 0}, // never ran
	}
	// Node 2's config drifted; node 1 unchanged.
	w := wsForReuse(t, nodes, "/g/genesis.json", "gh", map[int]string{1: "h1", 2: "h2-new"})
	snap := reuseSnapshot{
		genesisHash: "gh",
		before: map[int]nodeBaseline{
			1: {Index: 1, ConfigHash: "h1", Binary: "/bin/gwbft", PID: 111},
			2: {Index: 2, ConfigHash: "h2", Binary: "/bin/gwbft", PID: 0},
		},
		alive: map[int]bool{1: true},
	}
	plan, err := w.reconcileReuse(context.Background(), snap)
	if err != nil {
		t.Fatalf("reconcile: %v", err)
	}
	if got := plan.redo(); len(got) != 1 || got[0] != 2 {
		t.Fatalf("redo = %v, want [2]", got)
	}
	if w.state.Nodes[0].PID != 111 {
		t.Fatalf("reused node1 pid = %d, want 111", w.state.Nodes[0].PID)
	}
	if w.state.Nodes[1].PID != 0 {
		t.Fatalf("redo node2 pid = %d, want 0", w.state.Nodes[1].PID)
	}
}
