package local

import "context"

// StartNode invokes `chainbench.sh node start <nodeNum>`. nodeNum is the
// numeric pids.json key ("1", "2", ...). Thin wrapper over Run with a
// fixed argv shape — it performs no input validation; callers are
// responsible for checking the node exists and is currently stopped.
func (d *Driver) StartNode(ctx context.Context, nodeNum string) (*RunResult, error) {
	return d.Run(ctx, "node", "start", nodeNum)
}
