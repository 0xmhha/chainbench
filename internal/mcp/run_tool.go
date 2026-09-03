package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
)

// runTool runs DSL test specs and reports the session verdict — the MCP
// counterpart of the CLI `run`. Like the CLI it has two modes: with "rpc" it
// attaches to a running network (app.AttachRun); without it, it composes the
// network the specs' env declares and runs against it (app.RunSuite), so an
// agent reaches the same compose-and-run workflow the operator does.
func runTool() Tool {
	return Tool{
		Name: "chainbench_run",
		Description: "Run DSL test specs and report the verdict. With rpc (array of RPC URLs) it attaches to a running network; without rpc it composes the network the specs declare and runs against it. " +
			"Args: spec (a spec JSON string) and/or specs (array); rpc + chain for attach; dataDir/binary/validators/keysDir for compose.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"chain":      map[string]any{"type": "string"},
				"rpc":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"spec":       map[string]any{"type": "string"},
				"specs":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"dataDir":    map[string]any{"type": "string"},
				"binary":     map[string]any{"type": "string"},
				"validators": map[string]any{"type": "integer"},
				"keysDir":    map[string]any{"type": "string"},
			},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			specs := collectSpecs(args)
			if len(specs) == 0 {
				return "", fmt.Errorf("chainbench_run: provide spec or specs")
			}
			if rpcURLs := argStrings(args, "rpc"); len(rpcURLs) > 0 {
				return runAttach(ctx, argString(args, "chain", ""), rpcURLs, specs)
			}
			return runCompose(ctx, args, specs)
		},
	}
}

// runAttach runs the specs against an already-running network.
func runAttach(ctx context.Context, chain string, rpcURLs []string, specs [][]byte) (string, error) {
	if chain == "" {
		return "", fmt.Errorf("chainbench_run: chain is required to attach")
	}
	artifactRoot, err := os.MkdirTemp("", "cb-run")
	if err != nil {
		return "", fmt.Errorf("chainbench_run: temp dir: %w", err)
	}
	root, err := app.AttachRun(ctx, app.Deps{}, app.AttachRunIn{
		Chain: chain, RPCURLs: rpcURLs, ArtifactRoot: artifactRoot, Specs: specs,
	})
	if err != nil {
		return "", err
	}
	return formatRunSummary(root)
}

// runCompose composes the network the specs declare and runs against it,
// reaching the same app.RunSuite the CLI `run` does.
func runCompose(ctx context.Context, args map[string]any, specs [][]byte) (string, error) {
	dataDir := argString(args, "dataDir", "")
	if dataDir == "" {
		var err error
		if dataDir, err = os.MkdirTemp("", "cb-net"); err != nil {
			return "", fmt.Errorf("chainbench_run: temp dir: %w", err)
		}
	}
	out, err := app.RunSuite(ctx, app.Deps{}, app.RunSuiteIn{
		SpecContent: specs,
		DataDir:     dataDir,
		Chain:       argString(args, "chain", ""),
		Binary:      argString(args, "binary", ""),
		Validators:  argInt(args, "validators", 0),
		KeysDir:     argString(args, "keysDir", ""),
	})
	if err != nil {
		return "", err
	}
	if out.SessionRoot == "" {
		return "no session produced", nil
	}
	return formatRunSummary(out.SessionRoot)
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
	doc, err := app.SessionSummary(root)
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
