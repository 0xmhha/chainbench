package mcp

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/app"
)

// stopTool stops the nodes of a composed network by the pids its workspace
// records. It calls the same app.NetworkStop the CLI stop command does, so an
// agent tearing a network down after a test run gets identical behaviour.
func stopTool() Tool {
	return Tool{
		Name:        "chainbench_stop",
		Description: "Stop a composed network's nodes (by the pids its workspace records). Args: workspaceDir.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"workspaceDir": map[string]any{"type": "string"}},
			"required":   []string{"workspaceDir"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir := argString(args, "workspaceDir", "")
			if dir == "" {
				return "", fmt.Errorf("workspaceDir is required")
			}
			res, err := app.NetworkStop(ctx, app.Deps{}, app.NetworkStopIn{DataDir: dir})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("stopped %d node(s)", res.Stopped), nil
		},
	}
}
