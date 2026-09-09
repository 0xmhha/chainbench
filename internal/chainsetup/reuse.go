package chainsetup

import "fmt"

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
