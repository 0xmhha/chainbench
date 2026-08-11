package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/netcompose"
)

// Net step tools — the MCP mirrors of `chainbench net <step>`. Each handler is
// schema decoding + one app-layer call: the same function the CLI subcommand
// binds, so the two surfaces cannot drift (worklist T7.5).

// netNewTool initializes a composition workspace.
func netNewTool() Tool {
	return Tool{
		Name:        "chainbench_net_new",
		Description: "Initialize a step-composition workspace for a chain: records the chain, key set, and where the data plane lives (local or a remote SSH host).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"dataDir":    map[string]any{"type": "string", "description": "local workspace directory"},
				"chain":      map[string]any{"type": "string", "description": "chain id (stablenet|wbft|wemix)"},
				"binary":     map[string]any{"type": "string", "description": "node binary path (may also be set at start)"},
				"keys":       map[string]any{"type": "string", "description": "key set directory (default keys/preset)"},
				"remoteHost": map[string]any{"type": "string", "description": "SSH host for a remote data plane (empty = local)"},
				"remoteUser": map[string]any{"type": "string", "description": "SSH user"},
				"remotePort": map[string]any{"type": "number", "description": "SSH port (default 22)"},
				"targetDir":  map[string]any{"type": "string", "description": "data root ON the target"},
			},
			"required": []string{"dataDir", "chain"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetNew(ctx, app.Deps{}, app.NetNewIn{
				DataDir: argString(args, "dataDir", ""),
				Chain:   argString(args, "chain", ""),
				Binary:  argString(args, "binary", ""),
				KeysDir: argString(args, "keys", ""),
				Target:  targetSpecFromArgs(args),
			})
			if err != nil {
				return "", err
			}
			return out.Detail, nil
		},
	}
}

// netStatusTool reports the workspace composition state.
func netStatusTool() Tool {
	return Tool{
		Name:        "chainbench_net_status",
		Description: "Show a step-composition workspace's state as JSON: chain, target, and which steps have run.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"dataDir": map[string]any{"type": "string", "description": "local workspace directory"},
			},
			"required": []string{"dataDir"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			res, err := app.NetStatus(ctx, app.Deps{}, app.NetStatusIn{
				DataDir: argString(args, "dataDir", ""),
			})
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(res.State, "", "  ")
			if err != nil {
				return "", fmt.Errorf("mcp: net status: %w", err)
			}
			return string(b), nil
		},
	}
}

// targetSpecFromArgs maps the shared remote-target arguments onto a TargetSpec:
// remote when remoteHost is set, else local. Mirrors the CLI's targetFlags.
func targetSpecFromArgs(args map[string]any) netcompose.TargetSpec {
	host := argString(args, "remoteHost", "")
	if host == "" {
		return netcompose.TargetSpec{Kind: netcompose.TargetLocal, DataRoot: argString(args, "targetDir", "")}
	}
	return netcompose.TargetSpec{
		Kind: netcompose.TargetRemote, Host: host,
		User: argString(args, "remoteUser", ""), Port: argInt(args, "remotePort", 0),
		DataRoot: argString(args, "targetDir", ""),
	}
}
