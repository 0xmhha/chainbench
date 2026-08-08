package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/internal/engine"
)

// runTool runs DSL test specs through the redesign engine in attach mode against
// a running network's RPC endpoints, and reports the session verdict. It is the
// MCP counterpart of the CLI `run` command.
func runTool() Tool {
	return Tool{
		Name: "chainbench_run",
		Description: "Run DSL test specs against a running network (attach mode) and report the verdict. " +
			"Args: chain, rpc (array of RPC URLs), spec (a spec JSON string) and/or specs (array of spec JSON strings).",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain": map[string]any{"type": "string"},
				"rpc":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"spec":  map[string]any{"type": "string"},
				"specs": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"chain", "rpc"},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			chain := argString(args, "chain", "")
			rpcURLs := argStrings(args, "rpc")
			if chain == "" || len(rpcURLs) == 0 {
				return "", fmt.Errorf("chainbench_run: chain and rpc are required")
			}
			specs := collectSpecs(args)
			if len(specs) == 0 {
				return "", fmt.Errorf("chainbench_run: provide spec or specs")
			}

			artifactRoot, err := os.MkdirTemp("", "cb-run")
			if err != nil {
				return "", fmt.Errorf("chainbench_run: temp dir: %w", err)
			}
			eng, err := engine.NewAttachEngine(engine.AttachConfig{
				Chain: chain, RPCURLs: rpcURLs, ArtifactRoot: artifactRoot,
			})
			if err != nil {
				return "", err
			}
			root, err := eng.Run(ctx, specs)
			if err != nil {
				return "", err
			}
			return formatRunSummary(root)
		},
	}
}

// collectSpecs gathers spec JSON blobs from the "specs" array and the single
// "spec" string argument.
func collectSpecs(args map[string]any) [][]byte {
	var specs [][]byte
	for _, s := range argStrings(args, "specs") {
		if s != "" {
			specs = append(specs, []byte(s))
		}
	}
	if s := argString(args, "spec", ""); s != "" {
		specs = append(specs, []byte(s))
	}
	return specs
}

// formatRunSummary renders the session verdict as agent-readable text.
func formatRunSummary(root string) (string, error) {
	doc, err := engine.ReadSessionSummary(root)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, t := range doc.Tests {
		fmt.Fprintf(&b, "%d %s %s\n", t.Seq, t.ID, t.Status)
	}
	fmt.Fprintf(&b, "pass=%d fail=%d blocked=%d skip=%d",
		doc.Summary.Pass, doc.Summary.Fail, doc.Summary.Blocked, doc.Summary.Skip)
	return b.String(), nil
}
