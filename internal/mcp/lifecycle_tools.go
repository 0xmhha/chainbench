package mcp

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/config"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/state"
)

// startTool provisions and launches a local chain network, then persists its
// nodeset. It exposes the same core setup.Launch the CLI's `setup --launch`
// uses, so an agent can spin a network up (and later stop it with
// chainbench_stop). It needs a built node binary path.
func startTool() Tool {
	return Tool{
		Name: "chainbench_start",
		Description: "Provision and launch a local chain network, saving its nodeset. " +
			"Args: chain, binary, data_dir; optional validators, endpoints, keys_dir.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain":      map[string]any{"type": "string"},
				"binary":     map[string]any{"type": "string"},
				"data_dir":   map[string]any{"type": "string"},
				"validators": map[string]any{"type": "integer"},
				"endpoints":  map[string]any{"type": "integer"},
				"keys_dir":   map[string]any{"type": "string"},
			},
			"required": []string{"chain", "binary", "data_dir"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			chain := argString(args, "chain", "")
			binary := argString(args, "binary", "")
			dataDir := argString(args, "data_dir", "")
			if chain == "" || binary == "" || dataDir == "" {
				return "", fmt.Errorf("chain, binary, and data_dir are required")
			}
			p, err := registry.Get(chain)
			if err != nil {
				return "", err
			}
			override := config.Values{}
			if v := argInt(args, "validators", 0); v > 0 {
				override["nodes.validators"] = strconv.Itoa(v)
			}
			if v := argInt(args, "endpoints", -1); v >= 0 {
				override["nodes.endpoints"] = strconv.Itoa(v)
			}
			ns, err := setup.Launch(ctx, setup.LaunchOptions{
				Plugin:   p,
				Config:   config.Resolve(nil, override),
				DataRoot: dataDir,
				Binary:   binary,
				KeysDir:  argString(args, "keys_dir", "keys/preset"),
			})
			if err != nil {
				return "", err
			}
			if err := state.SaveNodeSet(dataDir, ns); err != nil {
				return "", err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "launched %s: %d node(s)\n", chain, len(ns.Nodes))
			for _, n := range ns.Nodes {
				fmt.Fprintf(&b, "  node%d %s %s pid=%d\n", n.Index, n.Role, n.RPCURL, n.PID)
			}
			return strings.TrimRight(b.String(), "\n"), nil
		},
	}
}

// stopTool stops the nodes of a launched local network, reading their PIDs from
// the setup's nodeset.json and stopping each through the driver. It calls the
// same app.NetworkStop the CLI stop command does, so an agent tearing a network
// down after a test run gets identical behaviour.
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
			res, err := app.NetworkStop(ctx, app.Deps{}, app.NetworkStopIn{DataDir: dir})
			if err != nil {
				return "", err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "stopped %d node(s)", res.Stopped)
			for _, e := range res.Failed {
				fmt.Fprintf(&b, "\n  %v", e)
			}
			return b.String(), nil
		},
	}
}
