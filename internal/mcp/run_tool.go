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
		Description: "Run DSL test specs and report the verdict. With rpc (array of RPC URLs) it attaches to a running network; with dataDir + attach it runs against the network that workspace already composed, using the capabilities it advertised; with dataDir alone it composes the network the specs declare. " +
			"Args: spec (a spec JSON string) and/or specs (array); rpc + chain for attach; dataDir + attach to run against a composed network; dataDir/binary/validators/keysDir for compose.",
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
				"attach": map[string]any{
					"type":        "boolean",
					"description": "the network in dataDir is already up: run against it, with the capabilities its composition advertised, instead of composing again",
				},
				"keepUp": map[string]any{
					"type":        "boolean",
					"description": "compose mode: leave the network running after the tests so it can be driven with follow-on rpc/tx/attach calls (default: stop it)",
				},
				"docker":     map[string]any{"type": "boolean"},
				"waitBlocks": map[string]any{"type": "integer"},
				"networkId":  map[string]any{"type": "integer"},
				"serverSet":  map[string]any{"type": "string", "description": "compose mode: server-set file selecting where the nodes run"},
				"server":     map[string]any{"type": "string", "description": "compose mode: server name within the server set"},
				"allServers": map[string]any{"type": "boolean", "description": "compose mode: spread the network across every server in the set"},
			},
		},
		Handler: func(ctx context.Context, args map[string]any) (string, error) {
			specs := collectSpecs(args)
			if len(specs) == 0 {
				return "", fmt.Errorf("chainbench_run: provide spec or specs")
			}
			if rpcURLs := argStrings(args, "rpc"); len(rpcURLs) > 0 {
				if argBool(args, "attach", false) {
					return "", fmt.Errorf("chainbench_run: attach takes the endpoints from dataDir; it does not combine with rpc")
				}
				return runAttach(ctx, argString(args, "chain", ""), rpcURLs, specs)
			}
			if argBool(args, "attach", false) {
				return runAttachWorkspace(ctx, argString(args, "dataDir", ""), argString(args, "chain", ""), specs)
			}
			return runCompose(ctx, args, specs)
		},
	}
}

// runAttachWorkspace runs the specs against the network a workspace composed,
// taking its endpoints and its advertised capabilities from the workspace.
//
// A spec that declares a capability it needs can only run this way: given
// endpoints alone the gate has nothing to check against and the spec skips.
func runAttachWorkspace(ctx context.Context, dataDir, chain string, specs [][]byte) (string, error) {
	if dataDir == "" {
		return "", fmt.Errorf("chainbench_run: attach needs dataDir, the workspace whose network is already up")
	}
	artifactRoot, err := os.MkdirTemp("", "cb-run")
	if err != nil {
		return "", fmt.Errorf("chainbench_run: temp dir: %w", err)
	}
	root, err := app.AttachRun(ctx, app.Deps{}, app.AttachRunIn{
		DataDir: dataDir, Chain: chain, ArtifactRoot: artifactRoot, Specs: specs,
	})
	if err != nil {
		return "", err
	}
	return formatRunSummary(root)
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
		// Parity with the CLI `run` flags (WA3): leave the network up for
		// follow-on dynamic actions, target a server set or docker, and wait for
		// the chain to reach a height before the specs run.
		KeepUp:     argBool(args, "keepUp", false),
		Docker:     argBool(args, "docker", false),
		WaitBlocks: uint64(argInt(args, "waitBlocks", 0)),
		NetworkID:  int64(argInt(args, "networkId", 0)),
		Server: app.ServerRef{
			SetPath: argString(args, "serverSet", ""),
			Name:    argString(args, "server", ""),
			All:     argBool(args, "allServers", false),
		},
	})
	if err != nil {
		return "", err
	}
	if out.SessionRoot == "" {
		return "no session produced", nil
	}
	text, rerr := formatRunSummary(out.SessionRoot)
	// When the network is left up, the workspace is the handle for follow-on
	// dynamic actions (rpc/tx/attach), so name it alongside the verdict — on the
	// error path too, since a failed run still leaves the network standing.
	if argBool(args, "keepUp", false) {
		note := fmt.Sprintf("\nnetwork: %s (left up)", dataDir)
		text += note
		if rerr != nil {
			rerr = fmt.Errorf("%s%s", rerr, note)
		}
	}
	return text, rerr
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

// formatRunSummary renders the session verdict as agent-readable text and
// signals failure. The text names the session root so a caller can fetch the
// full report with chainbench_report (WA6). A run with any failed or blocked
// test is returned as a tool error carrying that same text, so an automated
// caller that branches on the error does not read a failed run as success
// (WA7); a clean run returns it with no error.
func formatRunSummary(root string) (string, error) {
	doc, err := app.SessionSummary(root)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	for _, t := range doc.Tests {
		fmt.Fprintf(&b, "%d %s %s\n", t.Seq, t.ID, t.Status)
	}
	fmt.Fprintf(&b, "pass=%d fail=%d blocked=%d skip=%d\n",
		doc.Summary.Pass, doc.Summary.Fail, doc.Summary.Blocked, doc.Summary.Skip)
	fmt.Fprintf(&b, "session: %s", root)
	text := b.String()
	if doc.Summary.Fail > 0 || doc.Summary.Blocked > 0 {
		return text, fmt.Errorf("%s", text)
	}
	return text, nil
}
