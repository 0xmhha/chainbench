package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
	"github.com/0xmhha/chainbench/pkg/core/state"
	"github.com/0xmhha/chainbench/pkg/testkit"

	remotepkg "github.com/0xmhha/chainbench/pkg/core/remote"
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
			ns, err := state.LoadNetwork(stateDir, name)
			if err != nil {
				return "", err
			}
			n, ok := pickNode(ns, argInt(args, "node", 0))
			if !ok {
				return "", fmt.Errorf("network %q has no usable node", name)
			}
			hc, err := httpClientForNode(n)
			if err != nil {
				return "", err
			}
			var out json.RawMessage
			params, _ := args["params"].([]any)
			if err := rpc.DialWithClient(n.RPCURL, hc).Call(ctx, method, &out, params...); err != nil {
				return "", err
			}
			return string(out), nil
		},
	}
}

// httpClientForNode builds an http.Client that reaches a node through its stored
// auth descriptor (empty auth yields the default client). Shared by the tools
// that dial a saved network's nodes.
func httpClientForNode(n node.Node) (*http.Client, error) {
	return remotepkg.HTTPClientFromAuth(remotepkg.Auth(n.Auth), os.Getenv)
}

// pickNode returns the node with the given 1-based index, or the primary node
// when index <= 0.
func pickNode(ns node.NodeSet, index int) (node.Node, bool) {
	if index > 0 {
		for _, n := range ns.Nodes {
			if n.Index == index {
				return n, true
			}
		}
		return node.Node{}, false
	}
	return ns.Primary()
}

// testListTool lists the registered test cases, optionally filtered by category.
func testListTool() Tool {
	return Tool{
		Name:        "chainbench_test_list",
		Description: "List registered test cases. Args: optional category filter.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"category": map[string]any{"type": "string"}},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			cat := argString(args, "category", "")
			var b strings.Builder
			count := 0
			for _, c := range testkit.Cases() {
				if cat != "" && c.Category != cat {
					continue
				}
				fmt.Fprintf(&b, "%s\t%s\n", c.Name, c.Category)
				count++
			}
			if count == 0 {
				return "no test cases", nil
			}
			return strings.TrimRight(b.String(), "\n"), nil
		},
	}
}
