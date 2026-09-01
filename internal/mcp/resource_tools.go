package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/internal/app"
)

// netMapTool answers where nodes are, in both directions. It is the tool an
// agent reaches for instead of reading the workspace file and parsing it:
// "which node owns port 8610", "what runs on this host", "where is en2".
func chainShowTool() Tool {
	return Tool{
		Name: "chainbench_chain_show",
		Description: "Look up the composed network's placement. With no selector, the whole map; " +
			"with one, that question answered — including the reverse ones (which node owns a port, " +
			"what runs on an address). Each node has an identity (node7) and a role alias (en2).",
		InputSchema: workspaceDirSchema(map[string]any{
			"node":  map[string]any{"type": "number", "description": "select by identity (1-based node number)"},
			"label": map[string]any{"type": "string", "description": "select by identity (node7) or role alias (en2)"},
			"host":  map[string]any{"type": "string", "description": "select every node on an address"},
			"port":  map[string]any{"type": "number", "description": "select whichever node listens on a port (p2p, etcd, http, ws, auth, metrics)"},
			"addr":  map[string]any{"type": "string", "description": "select by an address as a log line prints it (host:port)"},
		}),
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			res, err := app.NetMap(ctx, app.Deps{}, app.NetMapIn{
				DataDir: argString(args, "workspaceDir", ""),
				Node:    argInt(args, "node", 0),
				Label:   argString(args, "label", ""),
				Host:    argString(args, "host", ""),
				Port:    argInt(args, "port", 0),
				Addr:    argString(args, "addr", ""),
			})
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return "", fmt.Errorf("mcp: chain show: %w", err)
			}
			return string(b), nil
		},
	}
}

// netPoolTool reports what a network may be composed from, so an agent sizing
// one can ask instead of guessing — and can explain a refusal.
//
// It returns no credentials, and that absence is fixed by a test: the pool says
// where nodes may run, and how to log in is not something an agent transcript
// should carry (the keyring's missing export tool is the same judgement).
func resourcePoolTool() Tool {
	return Tool{
		Name: "chainbench_resource_pool",
		Description: "Show the addresses and port slots a network may be composed from: hosts, slots per host, " +
			"total capacity, how many a workspace already uses, and where the port plan came from.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspaceDir": map[string]any{"type": "string", "description": "workspace to count used slots from (optional)"},
			},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			res, err := app.NetPool(ctx, app.Deps{}, app.NetPoolIn{
				DataDir: argString(args, "workspaceDir", ""),
			})
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return "", fmt.Errorf("mcp: resource pool: %w", err)
			}
			return string(b), nil
		},
	}
}

// resourcePlanTool runs the allocator as a question: the placement a network of
// this shape would get, from the server set (or the built-in pool), with
// nothing written anywhere.
func resourcePlanTool() Tool {
	return Tool{
		Name: "chainbench_resource_plan",
		Description: "Compute the placement a network shape would get, without composing anything: " +
			"deterministic host and port assignment for the requested validators and endpoints. " +
			"The chain sets the family's per-node port reservation.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain":      map[string]any{"type": "string", "description": "chain id (stablenet|wbft|wemix); default stablenet"},
				"validators": map[string]any{"type": "number", "description": "validator node count (default 4)"},
				"endpoints":  map[string]any{"type": "number", "description": "endpoint node count"},
				"serverSet": map[string]any{
					"type":        "string",
					"description": "server-set file (default: server-set.yaml when present)",
				},
				"server":      map[string]any{"type": "string", "description": "server to place nodes on, by name from the server set"},
				"all_servers": map[string]any{"type": "boolean", "description": "spread across every server in the server set"},
			},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			validators := argInt(args, "validators", 0)
			if validators == 0 {
				validators = 4
			}
			res, err := app.NetPlan(ctx, app.Deps{}, app.NetPlanIn{
				Chain:      argString(args, "chain", ""),
				Validators: validators,
				Endpoints:  argInt(args, "endpoints", 0),
				Server: app.ServerRef{
					SetPath: argString(args, "serverSet", ""),
					Name:    argString(args, "server", ""),
					All:     argBool(args, "all_servers", false),
				},
			})
			if err != nil {
				return "", err
			}
			b, err := json.MarshalIndent(res, "", "  ")
			if err != nil {
				return "", fmt.Errorf("mcp: resource plan: %w", err)
			}
			return string(b), nil
		},
	}
}
