package mcp

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/app"
)

// remoteRPCTool calls a JSON-RPC method on a saved attached network's endpoint,
// dialing through the node's stored auth (api-key / JWT). It is the synergy of
// the absorbed pieces: the named-network registry supplies the endpoint and auth
// descriptor, pkg/core/remote turns the descriptor into an authenticated
// http.Client, and the core rpc client makes the call.
func remoteRPCTool() Tool {
	return Tool{
		Name: "chainbench_remote_rpc",
		Description: "Call a JSON-RPC method on a saved attached network's endpoint, using its " +
			"stored auth. Args: name, state_dir, method; optional params (JSON array), node (1-based index).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":      map[string]any{"type": "string"},
				"state_dir": map[string]any{"type": "string"},
				"method":    map[string]any{"type": "string"},
				"params":    map[string]any{"type": "array"},
				"node":      map[string]any{"type": "integer"},
			},
			"required": []string{"name", "state_dir", "method"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argString(args, "name", "")
			stateDir := argString(args, "state_dir", "")
			method := argString(args, "method", "")
			if name == "" || stateDir == "" || method == "" {
				return "", fmt.Errorf("name, state_dir, and method are required")
			}
			ns, err := app.Network(app.Deps{}, stateDir, name)
			if err != nil {
				return "", err
			}
			n, ok := app.NodeAt(ns, argInt(args, "node", 0))
			if !ok {
				return "", fmt.Errorf("network %q has no usable node", name)
			}
			params, _ := args["params"].([]any)
			out, err := app.CallOnNode(ctx, app.Deps{}, n, method, params...)
			if err != nil {
				return "", err
			}
			return string(out), nil
		},
	}
}
