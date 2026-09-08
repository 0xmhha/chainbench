package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
)

// testListTool lists the DSL test cases under a directory so an agent can
// discover what is there before running one with chainbench_run. It is the MCP
// mirror of the CLI `test list`, over the same app.ListSpecs (WA2).
func testListTool() Tool {
	return Tool{
		Name:        "chainbench_test_list",
		ReadOnly:    true,
		Description: "List the runnable DSL test cases under a directory (recursively): path, id, chain, description. Args: dir.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"dir": map[string]any{"type": "string", "description": "directory to walk for test cases"},
			},
			"required": []string{"dir"},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			dir := argString(args, "dir", "")
			if dir == "" {
				return "", fmt.Errorf("chainbench_test_list: dir is required")
			}
			specs, err := app.ListSpecs(dir)
			if err != nil {
				return "", err
			}
			if len(specs) == 0 {
				return "no test cases found", nil
			}
			var b strings.Builder
			for _, s := range specs {
				chain := s.Chain
				if chain == "" {
					chain = "-"
				}
				fmt.Fprintf(&b, "%s\t%s\t%s\t%s\n", s.Path, s.ID, chain, s.Description)
			}
			return strings.TrimRight(b.String(), "\n"), nil
		},
	}
}
