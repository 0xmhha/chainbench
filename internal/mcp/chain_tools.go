package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Net step tools — the MCP mirrors of `chainbench net <step>`. Each handler is
// schema decoding + one app-layer call: the same function the CLI subcommand
// binds, so the two surfaces cannot drift.

// chainNewTool initializes a composition workspace.
func chainNewTool() Tool {
	return Tool{
		Name:        "chainbench_chain_new",
		Description: "Initialize a step-composition workspace for a chain: records the chain, key set, and where the data plane lives (local or a remote SSH host).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspaceDir": map[string]any{"type": "string", "description": "workspace directory (where the composition is set up)"},
				"chain":        map[string]any{"type": "string", "description": "chain id (stablenet|wbft|wemix)"},
				"binary":       map[string]any{"type": "string", "description": "node binary path (may also be set at start)"},
				"keys":         map[string]any{"type": "string", "description": "key set directory (default keys/preset)"},
				"target":       map[string]any{"type": "string", "description": "data plane as one path: /local/path | user@host:/path | ssh://user@host:port/path"},
				"remoteHost":   map[string]any{"type": "string", "description": "legacy: SSH host for a remote data plane (prefer target)"},
				"remoteUser":   map[string]any{"type": "string", "description": "legacy: SSH user (prefer target)"},
				"remotePort":   map[string]any{"type": "number", "description": "legacy: SSH port (prefer target)"},
				"targetDir":    map[string]any{"type": "string", "description": "legacy: data root ON the target (prefer target)"},
				"docker":       map[string]any{"type": "boolean", "description": "servers are local docker containers: translate this tool's dials via the localmap next to the server set"},
			},
			"required": []string{"workspaceDir", "chain"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			target, err := targetSpecFromArgs(args)
			if err != nil {
				return "", err
			}
			out, err := app.NetNew(ctx, app.Deps{}, app.NetNewIn{
				DataDir: argString(args, "workspaceDir", ""),
				Chain:   argString(args, "chain", ""),
				Binary:  argString(args, "binary", ""),
				KeysDir: argString(args, "keys", ""),
				Target:  target,
				Docker:  argBool(args, "docker", false),
			})
			if err != nil {
				return "", err
			}
			return out.Detail, nil
		},
	}
}

// chainStatusTool reports the workspace composition state.
func chainStatusTool() Tool {
	return Tool{
		Name:        "chainbench_chain_status",
		Description: "Show a step-composition workspace's state as JSON: chain, target, and which steps have run.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspaceDir": map[string]any{"type": "string", "description": "workspace directory (where the composition is set up)"},
			},
			"required": []string{"workspaceDir"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			res, err := app.NetStatus(ctx, app.Deps{}, app.NetStatusIn{
				DataDir: argString(args, "workspaceDir", ""),
			})
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(res.State, "", "  ")
			if err != nil {
				return "", fmt.Errorf("mcp: chain status: %w", err)
			}
			return string(b), nil
		},
	}
}

// dataDirSchema is the one argument every step shares.
func workspaceDirSchema(extra map[string]any) map[string]any {
	props := map[string]any{
		"workspaceDir": map[string]any{"type": "string", "description": "workspace directory (where the composition is set up)"},
	}
	maps.Copy(props, extra)
	return map[string]any{"type": "object", "properties": props, "required": []string{"workspaceDir"}}
}

// chainKeysTool ensures the workspace's key set.
func chainKeysTool() Tool {
	return Tool{
		Name:        "chainbench_chain_keys",
		Description: "Ensure the workspace's key set exists and covers the node count (preset, or generate a fresh set in process).",
		InputSchema: workspaceDirSchema(map[string]any{
			"source":     map[string]any{"type": "string", "description": "preset (default) | generate"},
			"nodes":      map[string]any{"type": "number", "description": "identities the set must cover (default: allocated node count)"},
			"validators": map[string]any{"type": "number", "description": "identities joining the validator set (generate; 0 = all)"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetKeys(ctx, app.Deps{}, app.NetKeysIn{
				DataDir: argString(args, "workspaceDir", ""), Source: argString(args, "source", ""),
				Nodes: argInt(args, "nodes", 0), Validators: argInt(args, "validators", 0),
			})
			return out.Detail, err
		},
	}
}

// chainPlaceTool builds the node table.
func chainPlaceTool() Tool {
	return Tool{
		Name:        "chainbench_chain_place",
		Description: "Build the workspace's node table: roles, target-side paths, deterministic ports.",
		InputSchema: workspaceDirSchema(map[string]any{
			"validators": map[string]any{"type": "number", "description": "validator node count (default 4)"},
			"endpoints":  map[string]any{"type": "number", "description": "endpoint node count"},
			"peering": map[string]any{
				"type":        "string",
				"enum":        []string{"mesh", "proxied"},
				"description": "peer graph: mesh (default, every node dials every other) or proxied (bp <-> pn <-> en; endpoints never dial a producer)",
			},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetAllocate(ctx, app.Deps{}, app.NetAllocateIn{
				DataDir:    argString(args, "workspaceDir", ""),
				Validators: argInt(args, "validators", 4), Endpoints: argInt(args, "endpoints", 0),
				Peering: argString(args, "peering", ""),
			})
			return out.Detail, err
		},
	}
}

// chainGenesisTool builds the genesis.
func chainGenesisTool() Tool {
	return Tool{
		Name:        "chainbench_chain_genesis",
		Description: "Build the genesis from the key set and write it to the target (optionally overriding the chain id).",
		InputSchema: workspaceDirSchema(map[string]any{
			"chainId": map[string]any{"type": "number", "description": "override the manifest chain id (0 = manifest)"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetGenesis(ctx, app.Deps{}, app.NetGenesisIn{
				DataDir: argString(args, "workspaceDir", ""), ChainID: int64(argInt(args, "chainId", 0)),
			})
			return out.Detail, err
		},
	}
}

// chainConfigTool renders node configs.
func chainConfigTool() Tool {
	return Tool{
		Name:        "chainbench_chain_config",
		Description: "Render and write each node's TOML config to the target.",
		InputSchema: workspaceDirSchema(nil),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetConfig(ctx, app.Deps{}, app.NetConfigIn{DataDir: argString(args, "workspaceDir", "")})
			return out.Detail, err
		},
	}
}

// chainBuildTool assembles the launch commands.
func chainBuildTool() Tool {
	return Tool{
		Name:        "chainbench_chain_build",
		Description: "Assemble each node's launch command (without running it), optionally applying key=value overrides through the launchopt Builder.",
		InputSchema: workspaceDirSchema(map[string]any{
			"set": map[string]any{"type": "array", "items": map[string]any{"type": "string"},
				"description": "high-precedence launch knobs key=value (bare key for booleans)"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetLaunchOpts(ctx, app.Deps{}, app.NetLaunchOptsIn{
				DataDir: argString(args, "workspaceDir", ""), Set: argStrings(args, "set"),
			})
			if err != nil {
				return "", err
			}
			var b strings.Builder
			b.WriteString(out.Detail)
			for _, ns := range out.Nodes {
				fmt.Fprintf(&b, "\nnode%d: %s", ns.Index, strings.Join(ns.Args, " "))
			}
			return b.String(), nil
		},
	}
}

// chainDeployTool verifies launch inputs.
func chainDeployTool() Tool {
	return Tool{
		Name:        "chainbench_chain_deploy",
		Description: "Verify the launch inputs (genesis, configs) are present on the target; present files are reused, missing ones are named.",
		InputSchema: workspaceDirSchema(nil),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetProvision(ctx, app.Deps{}, app.NetProvisionIn{DataDir: argString(args, "workspaceDir", "")})
			return out.Detail, err
		},
	}
}

// chainInitTool initializes datadirs.
func chainInitTool() Tool {
	return Tool{
		Name:        "chainbench_chain_init",
		Description: "Initialize each node's datadir from the built genesis (runs `<binary> init`).",
		InputSchema: workspaceDirSchema(map[string]any{
			"binary": map[string]any{"type": "string", "description": "node binary path (default: the workspace's)"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetInit(ctx, app.Deps{}, app.NetInitIn{
				DataDir: argString(args, "workspaceDir", ""), Binary: argString(args, "binary", ""),
			})
			return out.Detail, err
		},
	}
}

// chainStartTool launches the network.
func chainStartTool() Tool {
	return Tool{
		Name:        "chainbench_chain_start",
		Description: "Launch every stopped node of the composed network and record its PID.",
		InputSchema: workspaceDirSchema(map[string]any{
			"binary": map[string]any{"type": "string", "description": "node binary path (default: the workspace's)"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetStart(ctx, app.Deps{}, app.NetStartIn{
				DataDir: argString(args, "workspaceDir", ""), Binary: argString(args, "binary", ""),
			})
			return out.Detail, err
		},
	}
}

// chainStopTool stops the network.
func chainStopTool() Tool {
	return Tool{
		Name:        "chainbench_chain_stop",
		Description: "Stop every running node by its recorded PID.",
		InputSchema: workspaceDirSchema(nil),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetStop(ctx, app.Deps{}, app.NetStopIn{DataDir: argString(args, "workspaceDir", "")})
			return out.Detail, err
		},
	}
}

// chainRestartTool bounces one node.
func chainRestartTool() Tool {
	return Tool{
		Name:        "chainbench_chain_restart",
		Description: "Stop and relaunch one node with its recorded arming (the exact argv it started with).",
		InputSchema: workspaceDirSchema(map[string]any{
			"node": map[string]any{"type": "number", "description": "node index (1-based)"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetRestart(ctx, app.Deps{}, app.NetRestartIn{
				DataDir: argString(args, "workspaceDir", ""), Node: argInt(args, "node", 0),
			})
			return out.Detail, err
		},
	}
}

// chainRmTool removes the data plane.
func chainRmTool() Tool {
	return Tool{
		Name:        "chainbench_chain_rm",
		Description: "Remove the composed data plane (node datadirs, configs, genesis). Running nodes must be stopped first.",
		InputSchema: workspaceDirSchema(nil),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetRm(ctx, app.Deps{}, app.NetRmIn{DataDir: argString(args, "workspaceDir", "")})
			return out.Detail, err
		},
	}
}

// chainLogsTool tails one node's log.
func chainLogsTool() Tool {
	return Tool{
		Name:        "chainbench_chain_logs",
		Description: "Show the last lines of one node's log.",
		InputSchema: workspaceDirSchema(map[string]any{
			"node":  map[string]any{"type": "number", "description": "node index (1-based)"},
			"lines": map[string]any{"type": "number", "description": "lines from the end (default 50)"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetLogs(ctx, app.Deps{}, app.NetLogsIn{
				DataDir: argString(args, "workspaceDir", ""),
				Node:    argInt(args, "node", 0), Lines: argInt(args, "lines", 50),
			})
			return out.Text, err
		},
	}
}

// chainHealthTool probes the nodes.
func chainHealthTool() Tool {
	return Tool{
		Name:        "chainbench_chain_health",
		Description: "Probe every node's HTTP RPC for its latest block height; returns a JSON table.",
		InputSchema: workspaceDirSchema(nil),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetHealth(ctx, app.Deps{}, app.NetHealthIn{DataDir: argString(args, "workspaceDir", "")})
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(out.Nodes, "", "  ")
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	}
}

// targetSpecFromArgs maps the target arguments onto a TargetSpec. The
// single-path "target" argument wins and cannot be mixed with the legacy
// four-argument form. Mirrors the CLI's targetFlags.
func targetSpecFromArgs(args map[string]any) (resource.Spec, error) {
	host := argString(args, "remoteHost", "")
	if t := argString(args, "target", ""); t != "" {
		if host != "" || argString(args, "remoteUser", "") != "" ||
			argInt(args, "remotePort", 0) != 0 || argString(args, "targetDir", "") != "" {
			return resource.Spec{}, fmt.Errorf(
				"mcp: target and the legacy remoteHost/remoteUser/remotePort/targetDir arguments cannot be mixed")
		}
		return resource.Parse(t)
	}
	if host == "" {
		return resource.Spec{DataRoot: argString(args, "targetDir", "")}, nil
	}
	return resource.Spec{
		Host: host,
		User: argString(args, "remoteUser", ""), Port: argInt(args, "remotePort", 0),
		DataRoot: argString(args, "targetDir", ""),
	}, nil
}

// chainResumeTool recovers a workspace whose run died.
func chainResumeTool() Tool {
	return Tool{
		Name:        "chainbench_chain_resume",
		Description: "Recover a workspace whose run died: reconcile recorded pids with the machine, continue the composition from the first unfinished step, and bring back nodes that should be running.",
		InputSchema: workspaceDirSchema(map[string]any{
			"binary": map[string]any{"type": "string", "description": "node binary path (default: the one the workspace recorded)"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			out, err := app.NetResume(ctx, app.Deps{}, app.NetResumeIn{
				DataDir: argString(args, "workspaceDir", ""), Binary: argString(args, "binary", ""),
			})
			var b strings.Builder
			for _, line := range out.Reconciled {
				fmt.Fprintf(&b, "reconcile: %s\n", line)
			}
			if out.Resumed != "" {
				fmt.Fprintf(&b, "resumed from: %s\n", out.Resumed)
			}
			for _, step := range out.Steps {
				fmt.Fprintln(&b, step)
			}
			for _, s := range out.Started {
				fmt.Fprintf(&b, "started: %s\n", s)
			}
			if err != nil {
				return b.String(), err
			}
			for _, n := range out.Nodes.Nodes.Nodes {
				fmt.Fprintf(&b, "node%d %s %s pid=%d\n", n.Index, n.Role, n.RPCURL, n.PID)
			}
			return strings.TrimRight(b.String(), "\n"), nil
		},
	}
}
