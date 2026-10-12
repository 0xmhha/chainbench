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

// nodeSwapTool relaunches one node on a different binary and/or configuration,
// the MCP mirror of `chainbench node swap`.
func nodeSwapTool() Tool {
	return Tool{
		Name:        "chainbench_node_swap",
		Description: "Relaunch one node of a composed network on a different binary and/or configuration, keeping its datadir and genesis. Args: workspaceDir, index, binary, config (key=value list), genesisOverlay (JSON text), args (flags appended to the command line), purpose.",
		InputSchema: workspaceDirSchema(map[string]any{
			"index":          map[string]any{"type": "integer", "description": "1-based node index to swap"},
			"binary":         map[string]any{"type": "string", "description": "binary to relaunch on (empty keeps the current one)"},
			"config":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "key=value config overrides applied before relaunch"},
			"genesisOverlay": map[string]any{"type": "string", "description": "genesis JSON fragment merged into this node's genesis"},
			"args":           map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "flags appended to the node's command line"},
			"purpose":        map[string]any{"type": "string", "description": "name recorded for the config fixture (config-<purpose>)"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			in := app.NodeSwapIn{
				DataDir: argString(args, "workspaceDir", ""), Index: argInt(args, "index", 0),
				Binary: argString(args, "binary", ""), Config: argStrings(args, "config"),
				Args: argStrings(args, "args"), Purpose: argString(args, "purpose", ""),
			}
			if overlay := argString(args, "genesisOverlay", ""); overlay != "" {
				in.GenesisOverlay = []byte(overlay)
			}
			res, err := app.NodeSwap(ctx, app.Deps{}, in)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("swapped node%d (pid %d)", in.Index, res.Node.PID), nil
		},
	}
}

// nodeResetTool resets one non-producing node to its genesis and leaves it
// stopped, the MCP mirror of `chainbench node reset`.
func nodeResetTool() Tool {
	return Tool{
		Name:        "chainbench_node_reset",
		Description: "Stop one non-producing node of a composed network, initialise its datadir again at genesis and leave it stopped. A block producer is refused. Args: workspaceDir, index.",
		InputSchema: nodeStopSchema(),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			index := argInt(args, "index", 0)
			if err := app.NodeReset(ctx, app.Deps{}, app.NodeResetIn{
				DataDir: argString(args, "workspaceDir", ""), Index: index,
			}); err != nil {
				return "", err
			}
			return fmt.Sprintf("node%d reset to its genesis and left stopped", index), nil
		},
	}
}
