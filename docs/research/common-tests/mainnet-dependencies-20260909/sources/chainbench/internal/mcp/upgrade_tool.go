package mcp

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/app"
)

// upgradeTool runs a profile-based consensus handoff (e.g. go-wemix -> go-wbft
// at a fork), the MCP counterpart of the CLI `upgrade run`, reaching the same
// app.UpgradeRun so an agent drives the identical sequence. It is distinct from
// a general per-node mixed-binary network, which is composed through
// chainbench_run from a spec's env.
func upgradeTool() Tool {
	return Tool{
		Name: "chainbench_upgrade",
		Description: "Run a profile-based consensus handoff (go-wemix -> go-wbft at a fork). " +
			"Args: profile (golden upgrade profile path), workspaceDir; optional preset, fromBinary, toBinary, template, genesisOverlay.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"profile":        map[string]any{"type": "string"},
				"workspaceDir":   map[string]any{"type": "string"},
				"preset":         map[string]any{"type": "string"},
				"fromBinary":     map[string]any{"type": "string"},
				"toBinary":       map[string]any{"type": "string"},
				"template":       map[string]any{"type": "string"},
				"genesisOverlay": map[string]any{"type": "string"},
			},
			"required": []string{"profile", "workspaceDir"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			profile := argString(args, "profile", "")
			dataDir := argString(args, "workspaceDir", "")
			if profile == "" || dataDir == "" {
				return "", fmt.Errorf("chainbench_upgrade: profile and workspaceDir are required")
			}
			out, err := app.UpgradeRun(ctx, app.Deps{}, app.UpgradeRunIn{
				ProfilePath:    profile,
				PresetDir:      argString(args, "preset", "keys/preset"),
				FromBinary:     argString(args, "fromBinary", ""),
				ToBinary:       argString(args, "toBinary", ""),
				Template:       argString(args, "template", ""),
				GenesisOverlay: argString(args, "genesisOverlay", ""),
				DataDir:        dataDir,
			})
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("handoff complete: %d node(s), governance %s, etcd cluster %q",
				len(out.Nodes.Nodes), out.Governance, out.Cluster), nil
		},
	}
}
