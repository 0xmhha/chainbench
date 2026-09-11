package chainsetup

import (
	"context"
	"os"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// reuse-if-matching's whole promise is that a node whose inputs did not move
// keeps running. The per-node verdict is tested, and so is the bookkeeping the
// verdict leaves behind, but nothing checked the one irreversible act in the
// middle: which process gets stopped. Every driver stub in this package returns
// nil from Stop and records nothing, so a reconciliation that stopped the whole
// network on any drift would pass all of them.
//
// The existing partial case (TestReconcileReuse_StoppedDriftedNodeClearedFromLedger)
// gives the drifted node PID 0 -- it never ran -- so it does not reach the stop at
// all.

// recordingDriver notes every Stop it is asked for.
type recordingDriver struct {
	stopped []int
}

func (d *recordingDriver) Provision(context.Context, process.NodeSpec) error { return nil }
func (d *recordingDriver) Launch(context.Context, process.NodeSpec) (process.Handle, error) {
	return process.Handle{}, nil
}
func (d *recordingDriver) Stop(_ context.Context, h process.Handle) error {
	d.stopped = append(d.stopped, h.PID)
	return nil
}

// TestReconcileReuse_StopsOnlyTheDriftedNodesProcess: one node drifted, both are
// up and answering, and only the drifted one may be stopped.
func TestReconcileReuse_StopsOnlyTheDriftedNodesProcess(t *testing.T) {
	const pidKept, pidDrifted = 111, 222
	nodes := []node.Record{
		{Index: 1, Label: "bp1", Server: "s1", ConfigPath: "/c/1.toml", PID: pidKept},
		{Index: 2, Label: "bp2", Server: "s1", ConfigPath: "/c/2.toml", PID: pidDrifted},
	}
	// The candidate for node2 differs from what it was launched with; node1's matches.
	w := wsForReuse(t, nodes, "/g/genesis.json", "gh", map[int]string{1: "h1", 2: "h2-new"})
	w.SetEnv(os.Getenv)
	drv := &recordingDriver{}
	w.machines = map[string]*resource.Access{
		"s1": {Spec: resource.Spec{Server: "s1"}, DataRoot: "/data/chainbench", Driver: drv},
	}
	snap := reuseSnapshot{
		genesisHash: "gh",
		before: map[int]nodeBaseline{
			1: {Index: 1, ConfigHash: "h1", Binary: "/bin/gwbft", PID: pidKept},
			2: {Index: 2, ConfigHash: "h2", Binary: "/bin/gwbft", PID: pidDrifted},
		},
		// Both are up and answering: the only thing separating them is the config.
		alive: map[int]bool{1: true, 2: true},
	}

	plan, err := w.reconcileReuse(context.Background(), snap, candFor(w))
	if err != nil {
		t.Fatalf("reconcileReuse: %v", err)
	}
	if plan.Refuse != "" {
		t.Fatalf("one drifted node must not refuse the whole reuse: %s", plan.Refuse)
	}
	if got := plan.redo(); len(got) != 1 || got[0] != 2 {
		t.Fatalf("redo = %v, want [2]", got)
	}

	// The irreversible half: exactly one process was stopped, and it was node2's.
	if len(drv.stopped) != 1 {
		t.Fatalf("stopped %v, want exactly node2's pid %d — a drift must not tear down the network", drv.stopped, pidDrifted)
	}
	if drv.stopped[0] != pidDrifted {
		t.Fatalf("stopped pid %d, want node2's %d", drv.stopped[0], pidDrifted)
	}
	// And the bookkeeping agrees with what was done.
	if w.state.Nodes[0].PID != pidKept {
		t.Errorf("reused node1 pid = %d, want %d kept", w.state.Nodes[0].PID, pidKept)
	}
	if w.state.Nodes[1].PID != 0 {
		t.Errorf("redone node2 pid = %d, want it cleared so init and start rework it", w.state.Nodes[1].PID)
	}
}

// TestReconcileReuse_StopsNothingWhenEveryNodeMatches: the common case must cost
// no restarts at all, which is the reason to have reuse-if-matching.
func TestReconcileReuse_StopsNothingWhenEveryNodeMatches(t *testing.T) {
	nodes := []node.Record{
		{Index: 1, Label: "bp1", Server: "s1", ConfigPath: "/c/1.toml", PID: 111},
		{Index: 2, Label: "bp2", Server: "s1", ConfigPath: "/c/2.toml", PID: 222},
	}
	w := wsForReuse(t, nodes, "/g/genesis.json", "gh", map[int]string{1: "h1", 2: "h2"})
	w.SetEnv(os.Getenv)
	drv := &recordingDriver{}
	w.machines = map[string]*resource.Access{
		"s1": {Spec: resource.Spec{Server: "s1"}, DataRoot: "/data/chainbench", Driver: drv},
	}
	snap := reuseSnapshot{
		genesisHash: "gh",
		before: map[int]nodeBaseline{
			1: {Index: 1, ConfigHash: "h1", Binary: "/bin/gwbft", PID: 111},
			2: {Index: 2, ConfigHash: "h2", Binary: "/bin/gwbft", PID: 222},
		},
		alive: map[int]bool{1: true, 2: true},
	}
	if _, err := w.reconcileReuse(context.Background(), snap, candFor(w)); err != nil {
		t.Fatalf("reconcileReuse: %v", err)
	}
	if len(drv.stopped) != 0 {
		t.Fatalf("stopped %v with nothing drifted", drv.stopped)
	}
}

// TestReconcileReuse_RefusalStopsNothing: a refusal is the case where the running
// network must be exactly as it was, and a stop is not undoable.
func TestReconcileReuse_RefusalStopsNothing(t *testing.T) {
	nodes := []node.Record{
		{Index: 1, Label: "bp1", Server: "s1", ConfigPath: "/c/1.toml", PID: 111},
	}
	w := wsForReuse(t, nodes, "/g/genesis.json", "gh-old", map[int]string{1: "h1"})
	w.SetEnv(os.Getenv)
	drv := &recordingDriver{}
	w.machines = map[string]*resource.Access{
		"s1": {Spec: resource.Spec{Server: "s1"}, DataRoot: "/data/chainbench", Driver: drv},
	}
	// The recorded genesis hash is what the candidate carries; the snapshot's is
	// the older one, so the shared genesis changed.
	snap := reuseSnapshot{
		genesisHash: "gh-before",
		before:      map[int]nodeBaseline{1: {Index: 1, ConfigHash: "h1", Binary: "/bin/gwbft", PID: 111}},
		alive:       map[int]bool{1: true},
	}
	plan, err := w.reconcileReuse(context.Background(), snap, candFor(w))
	if err != nil {
		t.Fatalf("reconcileReuse: %v", err)
	}
	if plan.Refuse == "" {
		t.Fatal("a changed genesis must refuse")
	}
	if len(drv.stopped) != 0 {
		t.Fatalf("a refusal stopped %v; nothing may be torn down when nothing is reconciled", drv.stopped)
	}
	if w.state.Nodes[0].PID != 111 {
		t.Errorf("a refusal cleared node1's pid (%d)", w.state.Nodes[0].PID)
	}
}
