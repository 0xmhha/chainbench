package driver

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Node-process control over an already-launched NodeSet. These live here — next
// to Driver and NodeSpec, the contract they drive — rather than in the legacy
// pipeline package that used to own them, so the surfaces that control a running
// network (app use cases, live-test fixtures) do not depend on it (worklist
// T7.11).
//
// They operate on the PIDs a NodeSet records, so they only reach nodes
// chainbench launched; an attached node (PID 0) is not ours to stop.

// StopNode stops the single node with the given 1-based index in ns through d.
// It errors if the index is not in the set or the node has no live PID (nothing
// chainbench launched to stop). Used to create a sync gap by taking one node
// offline while the rest of the network keeps producing blocks.
func StopNode(ctx context.Context, d Driver, ns node.NodeSet, index int) error {
	for _, n := range ns.Nodes {
		if n.Index != index {
			continue
		}
		if n.PID <= 0 {
			return fmt.Errorf("driver: node%d has no live PID to stop", index)
		}
		return d.Stop(ctx, Handle{Index: index, PID: n.PID})
	}
	return fmt.Errorf("driver: node%d not found in node set", index)
}

// RelaunchNode brings one stopped node back from its spec and returns the
// refreshed Node (new PID, same endpoints). The datadir persists across a stop,
// so no re-init is needed; the node rejoins its peers and re-syncs the blocks it
// missed. It re-provisions the config first (a harmless rewrite of the same
// bytes) so a remote driver reships it, mirroring the launch path.
func RelaunchNode(ctx context.Context, d Driver, spec NodeSpec) (node.Node, error) {
	if err := d.Provision(ctx, spec); err != nil {
		return node.Node{}, fmt.Errorf("driver: reprovision node%d: %w", spec.Index, err)
	}
	h, err := d.Launch(ctx, spec)
	if err != nil {
		return node.Node{}, fmt.Errorf("driver: relaunch node%d: %w", spec.Index, err)
	}
	return node.Node{
		Index:  spec.Index,
		Role:   spec.Role,
		Host:   spec.Host,
		RPCURL: fmt.Sprintf("http://%s:%d", spec.Host, spec.Ports.HTTP),
		Ports:  spec.Ports,
		PID:    h.PID,
	}, nil
}

// StopNodeSet stops every launched node (PID > 0) in ns through d. It is
// best-effort: a node that fails to stop is collected in errs and the rest still
// stop, so one dead PID does not block tearing down the network. It returns how
// many were stopped and the per-node errors.
func StopNodeSet(ctx context.Context, d Driver, ns node.NodeSet) (int, []error) {
	stopped := 0
	var errs []error
	for _, n := range ns.Nodes {
		if n.PID <= 0 {
			continue
		}
		if err := d.Stop(ctx, Handle{Index: n.Index, PID: n.PID}); err != nil {
			errs = append(errs, fmt.Errorf("node%d (pid %d): %w", n.Index, n.PID, err))
			continue
		}
		stopped++
	}
	return stopped, errs
}
