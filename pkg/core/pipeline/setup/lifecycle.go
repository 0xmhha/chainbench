package setup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/node"
)

// StopNodeSet stops every launched node (PID > 0) in ns through the driver. It
// is best-effort: a node that fails to stop is collected in errs and the rest
// still stop, so one dead PID does not block tearing down the network. It
// returns how many were stopped and the per-node errors. Shared by the stop CLI
// command and the stop MCP tool so both behave identically.
func StopNodeSet(ctx context.Context, d driver.Driver, ns node.NodeSet) (int, []error) {
	stopped := 0
	var errs []error
	for _, n := range ns.Nodes {
		if n.PID <= 0 {
			continue
		}
		if err := d.Stop(ctx, driver.Handle{Index: n.Index, PID: n.PID}); err != nil {
			errs = append(errs, fmt.Errorf("node%d (pid %d): %w", n.Index, n.PID, err))
			continue
		}
		stopped++
	}
	return stopped, errs
}
