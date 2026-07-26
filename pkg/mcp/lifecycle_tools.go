package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/setup"
	"github.com/0xmhha/chainbench/pkg/core/state"
)

// stopTool stops the nodes of a launched local network, reading their PIDs from
// the setup's nodeset.json and stopping each through the driver. It exposes the
// same core StopNodeSet the CLI stop command uses, so an agent can tear a network
// down after a test run.
func stopTool() Tool {
	return Tool{
		Name:        "chainbench_stop",
		Description: "Stop a launched local network's nodes (by PID from nodeset.json). Args: data_dir.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"data_dir": map[string]any{"type": "string"}},
			"required":   []string{"data_dir"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dir := argString(args, "data_dir", "")
			if dir == "" {
				return "", fmt.Errorf("data_dir is required")
			}
			ns, err := state.LoadNodeSet(dir)
			if err != nil {
				return "", err
			}
			stopped, errs := setup.StopNodeSet(ctx, driver.NewLocalDriver(), ns)
			var b strings.Builder
			fmt.Fprintf(&b, "stopped %d node(s)", stopped)
			for _, e := range errs {
				fmt.Fprintf(&b, "\n  %v", e)
			}
			return b.String(), nil
		},
	}
}
