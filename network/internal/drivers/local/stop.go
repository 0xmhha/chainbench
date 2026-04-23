package local

import "context"

// StopNode invokes `chainbench.sh node stop <nodeNum>`. nodeNum is the
// numeric pids.json key ("1", "2", ...). This is a thin wrapper over Run
// with a fixed argv shape — it performs no input validation; callers are
// responsible for checking the node exists before invoking.
func (d *Driver) StopNode(ctx context.Context, nodeNum string) (*RunResult, error) {
	return d.Run(ctx, "node", "stop", nodeNum)
}
