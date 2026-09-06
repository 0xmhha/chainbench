package mcp

import (
	"context"
	"fmt"
	"github.com/0xmhha/chainbench/internal/app"
	"strings"
)

// networkTopologyTool reports each node's reachability and peer count for a saved
// network, giving a one-shot view of the mesh — the quickest way to see whether a
// multi-node network is fully connected or a node is isolated. It reads the named
// network from the registry, then probes every node's RPC (through its stored
// auth) for its peer count.
func networkTopologyTool() Tool {
	return Tool{
		Name:        "chainbench_network_topology",
		Description: "Per-node reachability and peer count for a saved network. Args: name, state_dir.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":      map[string]any{"type": "string"},
				"state_dir": map[string]any{"type": "string"},
			},
			"required": []string{"name", "state_dir"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argString(args, "name", "")
			stateDir := argString(args, "state_dir", "")
			if name == "" || stateDir == "" {
				return "", fmt.Errorf("name and state_dir are required")
			}
			ns, err := app.Network(app.Deps{}, stateDir, name)
			if err != nil {
				return "", err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "network=%s chain_type=%s nodes=%d\n", ns.Network, ns.Chain, len(ns.Nodes))
			up, down := 0, 0
			for _, n := range ns.Nodes {
				peers, err := nodePeerCount(ctx, n)
				if err != nil {
					down++
					fmt.Fprintf(&b, "  node%d %s down\n", n.Index, n.RPCURL)
					continue
				}
				up++
				fmt.Fprintf(&b, "  node%d %s up peers=%d\n", n.Index, n.RPCURL, peers)
			}
			fmt.Fprintf(&b, "up=%d down=%d", up, down)
			return b.String(), nil
		},
	}
}

// nodePeerCount asks a node how many peers it has, reached through whatever its
// record says is needed.
func nodePeerCount(ctx context.Context, n app.Node) (uint64, error) {
	return app.PeersOfNode(ctx, app.Deps{}, n)
}
