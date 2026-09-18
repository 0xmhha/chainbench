package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
)

// hardforkTool plans (and, with execute, performs) a chain hardfork: swapping
// the node binary at a fork block while keeping node data. It is the MCP
// counterpart of the CLI `hardfork`, reaching the same app.HardforkPlan /
// HardforkExecute — so an agent gets the plan and result the operator does.
// execute defaults false (plan only), mirroring the CLI's dry-run default.
func hardforkTool() Tool {
	return Tool{
		Name: "chainbench_hardfork",
		Description: "Plan (and optionally execute) a chain hardfork: swap the node binary at a fork block, keeping node data. " +
			"Args: workspaceDir, toChain, block (integer); optional toBinary, execute (default false = plan only).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"workspaceDir": map[string]any{"type": "string", "description": "workspace of the running from-chain"},
				"toChain":      map[string]any{"type": "string"},
				"toBinary":     map[string]any{"type": "string"},
				"block":        map[string]any{"type": "integer"},
				"execute":      map[string]any{"type": "boolean"},
			},
			"required": []string{"workspaceDir", "toChain"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			dataDir := argString(args, "workspaceDir", "")
			if dataDir == "" {
				return "", fmt.Errorf("chainbench_hardfork: workspaceDir is required")
			}
			toBinary := argString(args, "toBinary", "")
			planned, err := app.HardforkPlan(ctx, app.Deps{}, app.HardforkPlanIn{
				DataDir: dataDir, ToChain: argString(args, "toChain", ""),
				ToBinary: toBinary, Block: int64(argInt(args, "block", 0)),
			})
			if err != nil {
				return "", err
			}
			plan := planned.Plan

			var b strings.Builder
			fmt.Fprintf(&b, "hardfork: %s -> %s (binary %s -> %s) at block %d\n",
				plan.FromChain, plan.ToChain, plan.FromBinary, plan.ToBinary, plan.Block)
			for _, s := range plan.Swaps {
				fmt.Fprintf(&b, "  node%d %s -> %s\n", s.Index, s.FromBinary, s.ToBinary)
			}
			if !argBool(args, "execute", false) {
				fmt.Fprint(&b, "plan only (set execute=true to perform)")
				return b.String(), nil
			}

			bin, err := lookupNodeBinary(toBinary, planned.To.Manifest().Binary)
			if err != nil {
				return "", err
			}
			res, err := app.HardforkExecute(ctx, app.Deps{}, app.HardforkExecuteIn{
				Plan: planned, DataDir: dataDir, Binary: bin,
			})
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&b, "upgraded %d node(s) to %s (%s)", len(res.Nodes.Nodes), plan.ToChain, bin)
			return b.String(), nil
		},
	}
}

// lookupNodeBinary resolves the node binary to run: an explicit path if given,
// else the chain's manifest binary looked up on PATH.
func lookupNodeBinary(explicit, chainBinary string) (string, error) {
	name := explicit
	if name == "" {
		name = chainBinary
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("cannot find node binary %q: %w (build it or pass toBinary)", name, err)
	}
	return path, nil
}
