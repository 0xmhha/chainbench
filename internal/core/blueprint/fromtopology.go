package blueprint

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// FromTopology rewrites a node layout as a network declaration.
//
// A topology says a node's index, role, sync mode, bootnode flag and binary
// name. A blueprint says all of that and the keys, ports, server, accounts,
// alloc and governance besides, so the older format is a strict subset and the
// conversion loses nothing. That is what makes N6 an absorption rather than a
// migration with a lossy edge: a topology can be turned into the wider document
// and edited from there, and nobody has to hand-translate one.
//
// It exists so the transition can be finished without a flag day. Both formats
// are read for now; mixing them in one composition is refused, because two
// documents describing one layout means one of their authors is reading a
// network that is not theirs.
//
// Two fields have nowhere to land yet, and they are reported rather than
// dropped: the bootnode flag and per-node binary NAMES (a blueprint names paths
// directly). Silently losing either would produce a document that composes a
// different network from the file it came from, which is the failure this whole
// track exists to end.
func FromTopology(t node.Topology, in FromTopologyIn) (Blueprint, error) {
	sorted := t.Sorted()
	if len(sorted) == 0 {
		return Blueprint{}, fmt.Errorf("blueprint: from topology: the layout declares no nodes")
	}
	bp := Blueprint{Version: Version, Chain: t.Chain, Peering: in.Peering}
	if in.Binary != "" {
		bp.Binaries = &Binaries{Node: in.Binary}
	}
	ord := map[node.Role]int{}
	for _, e := range sorted {
		if e.Bootnode {
			return Blueprint{}, fmt.Errorf("blueprint: from topology: node %d is the bootnode, and a blueprint has no way to say that yet — convert by hand or keep the topology", e.Index)
		}
		if e.Binary != "" {
			return Blueprint{}, fmt.Errorf("blueprint: from topology: node %d names binary %q, and a blueprint names paths rather than topology binary names — give it as a binaries override", e.Index, e.Binary)
		}
		role := e.NodeRole()
		ord[role]++
		n := Node{Name: string(node.RoleLabel(role, ord[role])), Role: string(role)}
		// Only a mode the file actually chose is carried. Writing the default
		// out would turn an unstated value into a stated one, and the next
		// reader could not tell which the author meant.
		if e.SyncMode != "" {
			n.SyncMode = e.SyncMode
		}
		bp.Nodes = append(bp.Nodes, n)
	}
	if err := bp.Validate(); err != nil {
		return Blueprint{}, fmt.Errorf("blueprint: from topology: the converted document is not valid: %w", err)
	}
	return bp, nil
}

// FromTopologyIn carries what a topology does not say and a blueprint can.
type FromTopologyIn struct {
	// Binary is the node executable to record.
	Binary string
	// Peering is the peer graph to record; a topology does not carry one.
	Peering string
}
