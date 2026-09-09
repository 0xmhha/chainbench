package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
)

// Reuse reconciliation for execution.chain = reuse-if-matching.
//
// The judgment is per node, not per composition: of fifteen nodes, fourteen
// whose inputs and running state are unchanged are left running, and only the
// one that drifted is torn down and brought back. The one exception is a shared
// input — the genesis — which every node runs: a changed genesis is a different
// chain, so it cannot be reconciled node by node onto a network that is already
// up, and the whole reuse is refused rather than splitting the network.
//
// This file holds only the decision. It is a pure function over what each node
// hashed to before and after the compose steps re-ran, and whether each node is
// actually answering — so it is exhaustively testable without a target. The
// integration (snapshot, health probe, selective re-init/relaunch) calls it.

// nodeBaseline is what a node hashed to and how it ran, captured before an up
// re-runs the composition steps over an existing workspace.
type nodeBaseline struct {
	Index      int
	Label      string
	ConfigHash string
	Binary     string
	PID        int
}

// nodeTarget is what the freshly re-run compose steps produced for a node.
type nodeTarget struct {
	Index      int
	Label      string
	ConfigHash string
	Binary     string
}

// reuseDisposition is what the reconciliation decided for one node.
type reuseDisposition struct {
	Index int
	Label string
	Reuse bool
	// Reason says why a node must be redone; it is empty when Reuse is true.
	Reason string
}

// reusePlan is the whole reconciliation outcome for one up over an existing
// workspace.
type reusePlan struct {
	// Refuse is non-empty when the reuse cannot proceed at all — a changed
	// shared input — in which case no node is touched and the up stops. It
	// carries the reason for the operator.
	Refuse string
	// Nodes is the per-node decision, in the order of the freshly composed
	// table.
	Nodes []reuseDisposition
}

// redo lists the indices the plan says to tear down and bring back.
func (p reusePlan) redo() []int {
	var out []int
	for _, d := range p.Nodes {
		if !d.Reuse {
			out = append(out, d.Index)
		}
	}
	return out
}

// reused counts the nodes the plan leaves running.
func (p reusePlan) reused() int {
	n := 0
	for _, d := range p.Nodes {
		if d.Reuse {
			n++
		}
	}
	return n
}

// planReuse decides, node by node, what a reuse-if-matching up must do.
//
// genesisBefore/genesisAfter are the shared genesis hash as recorded before the
// steps re-ran and as they produced it now; an empty genesisBefore means this
// workspace had no prior genesis (a first up), which is not a change. before is
// the prior per-node baseline keyed by node index; alive reports whether each
// index is actually answering its RPC now. A node is reused only when its
// config and binary are unchanged AND it is running and answering; anything
// else — no prior node, changed config, changed binary, a dead or silent
// process — is redone.
func planReuse(genesisBefore, genesisAfter string, before map[int]nodeBaseline, after []nodeTarget, alive map[int]bool) reusePlan {
	if genesisBefore != "" && genesisAfter != "" && genesisBefore != genesisAfter {
		return reusePlan{Refuse: "genesis changed: reuse-if-matching cannot reconcile a running network onto a new chain — use execution.chain=fresh"}
	}
	plan := reusePlan{Nodes: make([]reuseDisposition, 0, len(after))}
	for _, t := range after {
		d := reuseDisposition{Index: t.Index, Label: t.Label}
		b, had := before[t.Index]
		switch {
		case !had || b.PID <= 0:
			d.Reason = "not running"
		case b.ConfigHash != t.ConfigHash:
			d.Reason = "config changed"
		case b.Binary != t.Binary:
			d.Reason = "binary changed"
		case !alive[t.Index]:
			d.Reason = "not answering"
		default:
			d.Reuse = true
		}
		plan.Nodes = append(plan.Nodes, d)
	}
	return plan
}

// reuseSnapshot is the running composition's baseline, captured before an up
// re-runs its steps and overwrites the record. planReuse compares it against
// what the re-run produces.
type reuseSnapshot struct {
	genesisHash string
	before      map[int]nodeBaseline
	alive       map[int]bool
}

// snapshotForReuse captures the prior per-node baseline and probes which nodes
// answer, before the compose steps re-run and reset the node table. An empty
// workspace (a first up) yields an empty snapshot, which planReuse reads as
// "compose every node fresh".
func (w *Workspace) snapshotForReuse(ctx context.Context) reuseSnapshot {
	snap := reuseSnapshot{before: map[int]nodeBaseline{}, alive: map[int]bool{}}
	if len(w.state.Nodes) == 0 {
		return snap
	}
	snap.genesisHash = w.state.LaunchInputs[w.state.GenesisPath]
	for _, ns := range w.state.Nodes {
		snap.before[ns.Index] = nodeBaseline{
			Index:      ns.Index,
			Label:      ns.Label,
			ConfigHash: w.state.LaunchInputs[ns.ConfigPath],
			Binary:     w.binaryFor(ns, w.state.Binary),
			PID:        ns.PID,
		}
	}
	// A probe error, or a node that does not answer, is a redo — never a reuse.
	// The probe is best-effort: it must not stop an up, only downgrade nodes.
	if h, err := w.Health(ctx); err == nil {
		for _, nh := range h {
			snap.alive[nh.Index] = nh.Err == ""
		}
	}
	return snap
}

// reconcileReuse decides, after the compose steps re-ran, which nodes to leave
// running and which to bring back, and carries out the teardown of the ones
// that changed. Reused nodes keep their pid so init and start skip them (their
// datadir and process are left untouched); a node that must be redone and is
// still up is stopped here so init can re-initialize its datadir. A changed
// shared genesis refuses the whole reuse and touches nothing.
func (w *Workspace) reconcileReuse(ctx context.Context, snap reuseSnapshot) (reusePlan, error) {
	genesisAfter := w.state.LaunchInputs[w.state.GenesisPath]
	after := make([]nodeTarget, 0, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		after = append(after, nodeTarget{
			Index:      ns.Index,
			Label:      ns.Label,
			ConfigHash: w.state.LaunchInputs[ns.ConfigPath],
			Binary:     w.binaryFor(ns, w.state.Binary),
		})
	}
	plan := planReuse(snap.genesisHash, genesisAfter, snap.before, after, snap.alive)
	if plan.Refuse != "" {
		return plan, nil
	}
	reuse := map[int]bool{}
	for _, d := range plan.Nodes {
		if d.Reuse {
			reuse[d.Index] = true
		}
	}
	for i := range w.state.Nodes {
		ns := &w.state.Nodes[i]
		b, had := snap.before[ns.Index]
		if reuse[ns.Index] {
			// Carry the running pid so init and start leave it alone.
			ns.PID = b.PID
			continue
		}
		// A node that must be redone starts from stopped: if its prior process
		// is still up, stop it before init re-initializes the datadir.
		if had && b.PID > 0 {
			if err := w.stopByPID(ctx, *ns, b.PID); err != nil {
				return plan, err
			}
		}
		ns.PID = 0
	}
	return plan, nil
}

// stopByPID stops a node's prior process by the pid the snapshot recorded, for
// the redo path where the re-run has already reset the node table's pids.
func (w *Workspace) stopByPID(ctx context.Context, ns node.Record, pid int) error {
	t, err := w.machineFor(ns)
	if err != nil {
		return err
	}
	if err := t.Driver.Stop(ctx, process.Handle{Index: ns.Index, PID: pid}); err != nil {
		return fmt.Errorf("chainsetup: reuse: stop node%d (pid %d): %w", ns.Index, pid, err)
	}
	return nil
}

// describe renders the plan as one line for a step's recorded detail.
func (p reusePlan) describe() string {
	if p.Refuse != "" {
		return "reuse refused: " + p.Refuse
	}
	redo := p.redo()
	if len(redo) == 0 {
		return fmt.Sprintf("reuse-if-matching: all %d node(s) match and are running; nothing to redo", len(p.Nodes))
	}
	return fmt.Sprintf("reuse-if-matching: %d node(s) reused, %d redone %v", p.reused(), len(redo), redo)
}
