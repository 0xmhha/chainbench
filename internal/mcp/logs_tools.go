package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/logs"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// logTimelineTool merges a setup's per-node logs into one chronological view, so
// a cross-node sequence (a consensus round, a handoff) can be read interleaved.
// Same filters as chainbench_log; the difference is the ordering.
func logTimelineTool() Tool {
	return Tool{
		Name: "chainbench_log_timeline",
		Description: "Merge per-node logs into one chronological timeline. Args: workspaceDir; " +
			"optional pattern, regexp (bool), node (int), level (min severity), limit (int).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspaceDir": map[string]any{"type": "string"},
				"pattern":      map[string]any{"type": "string"},
				"regexp":       map[string]any{"type": "boolean"},
				"node":         map[string]any{"type": "integer"},
				"level":        map[string]any{"type": "string"},
				"limit":        map[string]any{"type": "integer"},
			},
			"required": []string{"workspaceDir"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			dir := argString(args, "workspaceDir", "")
			if dir == "" {
				return "", fmt.Errorf("workspaceDir is required")
			}
			regexpMode, _ := args["regexp"].(bool)
			matches, err := logs.Timeline(dir, logs.SearchOpts{
				Pattern: argString(args, "pattern", ""),
				Regexp:  regexpMode,
				Node:    argInt(args, "node", 0),
				Level:   argString(args, "level", ""),
				Limit:   argInt(args, "limit", 0),
			})
			if err != nil {
				return "", err
			}
			if len(matches) == 0 {
				return "no matching log lines", nil
			}
			var b strings.Builder
			for _, m := range matches {
				fmt.Fprintf(&b, "node%d: %s\n", m.Node, m.Text)
			}
			fmt.Fprintf(&b, "%d line(s)", len(matches))
			return b.String(), nil
		},
	}
}

// networkPeersTool reports a node's peer count and, when the admin namespace is
// available, its connected peers' enodes. Useful for diagnosing peering (the
// most common local multi-node failure).
func networkPeersTool() Tool {
	return Tool{
		Name:        "chainbench_network_peers",
		Description: "Report a node's peer count and connected peers (if admin is enabled). Args: rpc.",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"rpc": map[string]any{"type": "string"}},
			"required":   []string{"rpc"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			rpcURL := argString(args, "rpc", "")
			if rpcURL == "" {
				return "", fmt.Errorf("rpc is required")
			}
			cli := rpc.Dial(rpcURL)
			count, err := cli.PeerCount(ctx)
			if err != nil {
				return "", err
			}
			var b strings.Builder
			fmt.Fprintf(&b, "peers=%d", count)
			// admin_peers is best-effort: absent when the admin namespace is off.
			var peers []struct {
				Enode   string `json:"enode"`
				Name    string `json:"name"`
				Network struct {
					RemoteAddress string `json:"remoteAddress"`
				} `json:"network"`
			}
			if err := cli.Call(ctx, "admin_peers", &peers); err == nil {
				for _, p := range peers {
					addr := p.Network.RemoteAddress
					if addr == "" {
						addr = p.Enode
					}
					fmt.Fprintf(&b, "\n  %s %s", addr, p.Name)
				}
			}
			return b.String(), nil
		},
	}
}
