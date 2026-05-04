package local

import "context"

// StartNode invokes `chainbench.sh node start <nodeNum> [--binary-path
// <binaryPath>]`. nodeNum is the numeric pids.json key ("1", "2", ...).
// binaryPath is optional — when non-empty it is appended as a `--binary-path
// <binaryPath>` flag pair, overriding the profile's chain.binary_path for
// this invocation.
//
// Thin wrapper over Run with a fixed argv shape — it performs no input
// validation; callers are responsible for checking the node exists, is
// currently stopped, and (if supplied) that binaryPath is absolute.
func (d *Driver) StartNode(ctx context.Context, nodeNum, binaryPath string) (*RunResult, error) {
	if binaryPath == "" {
		return d.Run(ctx, "node", "start", nodeNum)
	}
	return d.Run(ctx, "node", "start", nodeNum, "--binary-path", binaryPath)
}
