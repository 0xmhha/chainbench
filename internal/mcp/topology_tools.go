package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
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
			ns, err := loadNetwork(stateDir, name)
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

// nodePeerCount dials a node (through its stored auth) and returns its peer count.
func nodePeerCount(ctx context.Context, n node.Node) (uint64, error) {
	hc, err := httpClientForNode(n)
	if err != nil {
		return 0, err
	}
	return rpc.DialWithClient(n.RPCURL, hc).PeerCount(ctx)
}
