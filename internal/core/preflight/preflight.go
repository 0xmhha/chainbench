// Package preflight compares the chain composed on a target with the chain the
// next test needs, and says how much has to be rebuilt.
//
// Tests run in sequence on one server set, and each one arrives with a
// declared environment. Most of the time the network the previous test left
// behind is the network the next one wants, and rebuilding it from scratch
// costs minutes for nothing. Sometimes only one node differs — a different
// sync mode, a node moved to another server — and only that node needs to
// come down and up. Sometimes the chain, the keys or the genesis differ, and
// then everything does.
//
// Comparing the two declarations on paper is not enough: a network whose
// shape matches can still have a node that the last test left dead. So the
// comparison has two halves. Compare is the paper half — Have against Want,
// nothing touched. Check adds the live half: for every node the paper half
// would keep, a Liveness probe (the inspector's facts, a pid signal, an RPC
// head) says whether it is actually there, and a node that is not joins the
// rebuild list.
//
// This package judges; it does not look. What is on the target comes from the
// composition's record (chainsetup reads its workspace into Have) and from the
// probes the caller injects. It imports the node model and nothing that
// reaches a machine, so the verdict is testable on a table.
package preflight

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Node is one node as either side describes it: the facts that decide
// whether the composed node can serve the wanted one.
type Node struct {
	Index    int
	Role     node.Role
	SyncMode string
	// Server is the server-set entry the node runs on; empty means the
	// workspace's single target.
	Server string
	Host   string
	Ports  node.Endpoints
	// PID is the recorded process, 0 when stopped. Only Have carries it.
	PID int
}

// Have is the chain composed on the target now, read from the composition's
// record. Zero Nodes means nothing is composed.
type Have struct {
	Chain      string
	Binary     string
	KeysDir    string
	Peering    string
	ChainID    int64
	Validators int
	// GenesisSum identifies the genesis bytes on the target (any stable hash;
	// both sides must use the same one). Empty on either side skips the check.
	GenesisSum string
	Nodes      []Node
	// Started reports whether the composition reached the start step: a
	// network that was composed but never launched is a shape, not a chain.
	Started bool
}

// Want is the chain the next test needs, in the same terms.
type Want struct {
	Chain      string
	Binary     string
	KeysDir    string
	Peering    string
	ChainID    int64
	Validators int
	Endpoints  int
	GenesisSum string
	// Nodes, when given, pins per-node facts (sync mode, server). Empty means
	// the counts are the whole requirement.
	Nodes []Node
}

// Verdict is how much of the composed chain has to be rebuilt.
type Verdict int

const (
	// Reuse: the composed chain serves the wanted one as it stands.
	Reuse Verdict = iota
	// RebuildNodes: the shape is right; the nodes in Decision.Nodes have to
	// be brought down and up (a differing sync mode, a node the last test
	// left dead).
	RebuildNodes
	// RebuildAll: a network-wide fact differs — chain, keys, genesis, peering,
	// node count — and the composition has to be redone.
	RebuildAll
	// Compose: nothing is composed on the target yet.
	Compose
)

// String renders a verdict the way a report prints it.
func (v Verdict) String() string {
	switch v {
	case Reuse:
		return "reuse"
	case RebuildNodes:
		return "rebuild-nodes"
	case RebuildAll:
		return "rebuild-all"
	case Compose:
		return "compose"
	}
	return fmt.Sprintf("verdict(%d)", int(v))
}

// Decision is the verdict and why.
type Decision struct {
	Verdict Verdict
	// Nodes are the 1-based indices to rebuild when Verdict is RebuildNodes.
	Nodes []int
	// Reasons say what differed or what was found dead, one per line, so an
	// operator reading "rebuild-all" also reads which fact forced it.
	Reasons []string
}

// String renders the decision with its reasons.
func (d Decision) String() string {
	s := d.Verdict.String()
	if len(d.Nodes) > 0 {
		parts := make([]string, len(d.Nodes))
		for i, n := range d.Nodes {
			parts[i] = fmt.Sprintf("node%d", n)
		}
		s += " " + strings.Join(parts, ",")
	}
	if len(d.Reasons) > 0 {
		s += ": " + strings.Join(d.Reasons, "; ")
	}
	return s
}

// Compare is the paper half: Have against Want, nothing touched. Network-wide
// differences win over per-node ones — a chain that has to be recomposed is
// not also "node 3 needs a restart".
func Compare(have Have, want Want) Decision {
	if len(have.Nodes) == 0 {
		return Decision{Verdict: Compose, Reasons: []string{"nothing is composed on the target"}}
	}
	var all []string
	differs := func(what, a, b string) {
		if a != b {
			all = append(all, fmt.Sprintf("%s: have %q, want %q", what, a, b))
		}
	}
	differs("chain", have.Chain, want.Chain)
	if want.Binary != "" {
		differs("binary", have.Binary, want.Binary)
	}
	if want.KeysDir != "" {
		differs("keys", have.KeysDir, want.KeysDir)
	}
	if want.Peering != "" || have.Peering != "" {
		differs("peering", orMesh(have.Peering), orMesh(want.Peering))
	}
	if want.ChainID != 0 && have.ChainID != want.ChainID {
		all = append(all, fmt.Sprintf("chain id: have %d, want %d", have.ChainID, want.ChainID))
	}
	if have.GenesisSum != "" && want.GenesisSum != "" && have.GenesisSum != want.GenesisSum {
		all = append(all, "genesis differs")
	}
	// Want.Nodes pins facts about named nodes; it is not the node count. The
	// count is the counts, and a request that gives none accepts what is there.
	wantTotal := want.Validators + want.Endpoints
	if want.Validators != 0 && have.Validators != want.Validators {
		all = append(all, fmt.Sprintf("validators: have %d, want %d", have.Validators, want.Validators))
	}
	if wantTotal != 0 && len(have.Nodes) != wantTotal {
		all = append(all, fmt.Sprintf("nodes: have %d, want %d", len(have.Nodes), wantTotal))
	}
	if !have.Started {
		all = append(all, "the composition was never started")
	}
	if len(all) > 0 {
		return Decision{Verdict: RebuildAll, Reasons: all}
	}

	// Per-node: only what Want pins.
	var idx []int
	var reasons []string
	byIndex := map[int]Node{}
	for _, n := range have.Nodes {
		byIndex[n.Index] = n
	}
	for _, w := range want.Nodes {
		h, ok := byIndex[w.Index]
		if !ok {
			return Decision{Verdict: RebuildAll, Reasons: []string{fmt.Sprintf("node%d is wanted but not composed", w.Index)}}
		}
		var why []string
		if w.Role != "" && !node.Is(h.Role, w.Role) {
			why = append(why, fmt.Sprintf("role %s→%s", h.Role, w.Role))
		}
		if w.SyncMode != "" && h.SyncMode != w.SyncMode {
			why = append(why, fmt.Sprintf("sync %q→%q", h.SyncMode, w.SyncMode))
		}
		if w.Server != "" && h.Server != w.Server {
			why = append(why, fmt.Sprintf("server %q→%q", h.Server, w.Server))
		}
		if len(why) > 0 {
			idx = append(idx, w.Index)
			reasons = append(reasons, fmt.Sprintf("node%d: %s", w.Index, strings.Join(why, ", ")))
		}
	}
	if len(idx) > 0 {
		sort.Ints(idx)
		return Decision{Verdict: RebuildNodes, Nodes: idx, Reasons: reasons}
	}
	return Decision{Verdict: Reuse}
}

// Liveness answers whether a composed node is actually serving: the pid the
// record carries is alive and the node answers RPC. The reason, when it is
// not, names what failed so the decision can repeat it.
type Liveness func(ctx context.Context, n Node) (alive bool, reason string)

// Check is Compare plus the live half. Every node the paper comparison would
// keep is probed; a dead one joins the rebuild list. A network-wide verdict
// (RebuildAll, Compose) is returned as is — there is nothing to probe.
func Check(ctx context.Context, have Have, want Want, live Liveness) Decision {
	d := Compare(have, want)
	if d.Verdict == RebuildAll || d.Verdict == Compose || live == nil {
		return d
	}
	rebuild := map[int]bool{}
	for _, i := range d.Nodes {
		rebuild[i] = true
	}
	for _, n := range have.Nodes {
		if rebuild[n.Index] {
			continue
		}
		ok, why := live(ctx, n)
		if ok {
			continue
		}
		rebuild[n.Index] = true
		d.Reasons = append(d.Reasons, fmt.Sprintf("node%d: %s", n.Index, orText(why, "not alive")))
	}
	if len(rebuild) == 0 {
		return Decision{Verdict: Reuse}
	}
	idx := make([]int, 0, len(rebuild))
	for i := range rebuild {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	// Every node dead is not a restart; it is a network that has to come back
	// whole, which the composition knows how to do and a node loop does not.
	if len(idx) == len(have.Nodes) {
		return Decision{Verdict: RebuildAll, Reasons: append([]string{"no composed node is alive"}, d.Reasons...)}
	}
	return Decision{Verdict: RebuildNodes, Nodes: idx, Reasons: d.Reasons}
}

func orMesh(p string) string {
	if p == "" {
		return string(node.Mesh)
	}
	return p
}

func orText(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
