package mcp

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// networkAttachTool probes an RPC endpoint to identify the chain, then saves it
// as a named attached network in the state dir. It exposes the core probe +
// named-network registry (absorbed from the legacy network/ module) so an agent
// can attach to an already-running network by URL.
func networkAttachTool() Tool {
	return Tool{
		Name: "chainbench_network_attach",
		Description: "Probe an RPC endpoint and save it as a named attached network. " +
			"Args: name, rpc, state_dir; optional override (force chain type), auth (object).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":      map[string]any{"type": "string"},
				"rpc":       map[string]any{"type": "string"},
				"state_dir": map[string]any{"type": "string"},
				"override":  map[string]any{"type": "string"},
				"auth":      map[string]any{"type": "object"},
			},
			"required": []string{"name", "rpc", "state_dir"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			name := argString(args, "name", "")
			rpc := argString(args, "rpc", "")
			stateDir := argString(args, "state_dir", "")
			if name == "" || rpc == "" || stateDir == "" {
				return "", fmt.Errorf("name, rpc, and state_dir are required")
			}
			if !isValidNetworkName(name) {
				return "", fmt.Errorf("invalid network name %q (must match [a-z0-9][a-z0-9_-]* and not be 'local')", name)
			}
			res, err := collector.Detect(ctx, collector.Options{RPCURL: rpc, Override: argString(args, "override", "")})
			if err != nil {
				return "", err
			}
			n := node.Node{Index: 1, Role: node.RoleEndpoint, Host: hostOf(rpc), RPCURL: rpc}
			if a, ok := args["auth"].(map[string]any); ok && len(a) > 0 {
				n.Auth = a
			}
			ns := node.NodeSet{
				Chain: res.ChainType, Network: name,
				Nodes: []node.Node{n}, Capabilities: []string{"rpc"},
			}
			if err := saveNetwork(stateDir, ns); err != nil {
				return "", err
			}
			return fmt.Sprintf("attached %q: chain_type=%s chain_id=%d namespaces=%v",
				name, res.ChainType, res.ChainID, res.Namespaces), nil
		},
	}
}

// networkListTool lists the saved attached networks.
func networkListTool() Tool {
	return Tool{
		Name:        "chainbench_network_list",
		Description: "List saved attached networks. Args: state_dir.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"state_dir": map[string]any{"type": "string"}},
			"required":   []string{"state_dir"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			stateDir := argString(args, "state_dir", "")
			if stateDir == "" {
				return "", fmt.Errorf("state_dir is required")
			}
			nets, err := listNetworks(stateDir)
			if err != nil {
				return "", err
			}
			if len(nets) == 0 {
				return "no attached networks", nil
			}
			var b strings.Builder
			for _, ns := range nets {
				ep := ""
				if len(ns.Nodes) > 0 {
					ep = ns.Nodes[0].RPCURL
				}
				fmt.Fprintf(&b, "%s\t%s\t%s\n", ns.Network, ns.Chain, ep)
			}
			return strings.TrimRight(b.String(), "\n"), nil
		},
	}
}

// networkInfoTool reports a saved network's chain type and endpoints.
func networkInfoTool() Tool {
	return Tool{
		Name:        "chainbench_network_info",
		Description: "Show a saved attached network's nodes. Args: name, state_dir.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":      map[string]any{"type": "string"},
				"state_dir": map[string]any{"type": "string"},
			},
			"required": []string{"name", "state_dir"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
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
			for _, n := range ns.Nodes {
				authed := ""
				if len(n.Auth) > 0 {
					authed = " (auth)"
				}
				fmt.Fprintf(&b, "  node%d %s %s%s\n", n.Index, n.Role, n.RPCURL, authed)
			}
			return strings.TrimRight(b.String(), "\n"), nil
		},
	}
}

// networkDetachTool removes a saved attached network.
func networkDetachTool() Tool {
	return Tool{
		Name:        "chainbench_network_detach",
		Description: "Remove a saved attached network. Args: name, state_dir.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":      map[string]any{"type": "string"},
				"state_dir": map[string]any{"type": "string"},
			},
			"required": []string{"name", "state_dir"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			name := argString(args, "name", "")
			stateDir := argString(args, "state_dir", "")
			if name == "" || stateDir == "" {
				return "", fmt.Errorf("name and state_dir are required")
			}
			if err := removeNetwork(stateDir, name); err != nil {
				return "", err
			}
			return fmt.Sprintf("detached %q", name), nil
		},
	}
}

// hostOf returns the host of an RPC URL, or "" if it cannot be parsed.
func hostOf(rpc string) string {
	if u, err := url.Parse(rpc); err == nil {
		return u.Hostname()
	}
	return ""
}
