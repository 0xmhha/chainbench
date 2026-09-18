package mcp

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/app"
)

// Single-node lifecycle tools — the MCP mirrors of `chainbench node stop|start`.
// They let an agent kill and relaunch one node of a composed network to exercise
// sync-gap and failover behaviour dynamically, which chain_restart (a bounce)
// cannot express because it never leaves a node down (WA4).

// nodeStopSchema is the one shape both tools share: a workspace and a node index.
func nodeStopSchema() map[string]any {
	return workspaceDirSchema(map[string]any{
		"index": map[string]any{"type": "integer", "description": "1-based node index to act on"},
	})
}

// nodeStopTool stops a single node by index, leaving the rest of the network up.
func nodeStopTool() Tool {
	return Tool{
		Name:        "chainbench_node_stop",
		Description: "Stop a single node of a composed network by 1-based index, leaving the others running (for sync-gap/failover tests). Args: workspaceDir, index.",
		InputSchema: nodeStopSchema(),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			index := argInt(args, "index", 0)
			if err := app.NodeStop(ctx, app.Deps{}, app.NodeStopIn{
				DataDir: argString(args, "workspaceDir", ""), Index: index,
			}); err != nil {
				return "", err
			}
			return fmt.Sprintf("stopped node%d", index), nil
		},
	}
}

// nodeStartTool relaunches a single stopped node by index.
func nodeStartTool() Tool {
	return Tool{
		Name:        "chainbench_node_start",
		Description: "Relaunch a single stopped node of a composed network by 1-based index, recording its new PID. Args: workspaceDir, index.",
		InputSchema: nodeStopSchema(),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			res, err := app.NodeStart(ctx, app.Deps{}, app.NodeStartIn{
				DataDir: argString(args, "workspaceDir", ""), Index: argInt(args, "index", 0),
			})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("started node%d (pid %d)", res.Node.Index, res.Node.PID), nil
		},
	}
}
