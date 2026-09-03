package process

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// Node-process control over an already-launched NodeSet. It lives here — next to
// Driver and NodeSpec, the contract it drives — so the surfaces that tear a
// running network down (a handoff's teardown, live-test fixtures) do not depend
// on the legacy pipeline package that used to own it.
//
// It operates on the PIDs a NodeSet records, so it only reaches nodes chainbench
// launched; an attached node (PID 0) is not ours to stop.

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
