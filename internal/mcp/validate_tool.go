package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
)

// validateTool validates DSL specs offline — the same parse, name-resolution,
// selector, and capability checks the CLI `validate` runs — without composing
// or writing anything. It is the MCP counterpart of `chainbench validate`, so an
// agent gets the same verdict the operator would.
func validateTool() Tool {
	return Tool{
		Name: "chainbench_validate",
		Description: "Validate DSL test specs offline (no network, no writes) and report which are well-formed. " +
			"Args: spec (a spec JSON string) and/or specs (array of spec JSON strings); optional chain (also report applicability).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain": map[string]any{"type": "string"},
				"spec":  map[string]any{"type": "string"},
				"specs": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			specs := collectSpecs(args)
			if len(specs) == 0 {
				return "", fmt.Errorf("chainbench_validate: provide spec or specs")
			}
			labels := make([]string, len(specs))
			for i := range specs {
				labels[i] = fmt.Sprintf("spec[%d]", i)
			}
			results, err := app.ValidateContent(specs, labels, argString(args, "chain", ""))
			if err != nil {
				return "", err
			}
			var b strings.Builder
			invalid := 0
			for _, r := range results {
				id := r.ID
				if id == "" {
					id = "-"
				}
				fmt.Fprintf(&b, "%s %s %s\n", r.Spec, id, r.Result)
				if !r.OK {
					invalid++
				}
			}
			fmt.Fprintf(&b, "valid=%d invalid=%d", len(results)-invalid, invalid)
			return b.String(), nil
		},
	}
}
