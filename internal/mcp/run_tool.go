package mcp

import (
	"context"
	"encoding/json"
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
				"plan":            map[string]any{"type": "boolean", "description": "resolve the composed network without running it"},
				"suite":           map[string]any{"type": "boolean", "description": "run or plan several cases as one suite on one network; otherwise sequence definitions"},
				"binaryFor":       map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}, "description": "declared binary name to accepted executable path"},
				"readOnlyKeys":    map[string]any{"type": "boolean"},
				"noNodeControl":   map[string]any{"type": "boolean", "description": "attach without granting node control"},
				"serverLogins":    map[string]any{"type": "string"},
				"workspaceConfig": map[string]any{"type": "string"},
				"artifactRoot":    map[string]any{"type": "string"},
				"chain":           commonProperty("chain", nil),
				"rpc":             map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"spec":            map[string]any{"type": "string"},
				"specs":           map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				"dataDir":         map[string]any{"type": "string"},
				"binary":          map[string]any{"type": "string"},
				"bp":              map[string]any{"type": "integer", "description": "bp node count, overriding what the specs declare"},
				"keysDir":         map[string]any{"type": "string"},
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
			// The same four-way answer the CLI gets, from the same place.
			// This used to be two branches here with no third: a spec that
			// declared env.attach could not be run through this tool at all.
			rpcURLs := argStrings(args, "rpc")
			dataDir := argString(args, "dataDir", "")
			chain := argString(args, "chain", "")
			start, serr := app.StartFor(app.Named{
				Spell: app.ToolSpelling, RPCURLs: rpcURLs,
				Attach: argBool(args, "attach", false), WorkspaceDir: dataDir,
				Specs: func() ([][]byte, []string, error) { return specs, nil, nil },
			})
			if serr != nil {
				return "", fmt.Errorf("chainbench_run: %w", serr)
			}
			if start.Adopts() {
				if argBool(args, "plan", false) {
					return "", fmt.Errorf("plan is only available for composition")
				}
				return runAttached(ctx, args, start, chain, rpcURLs, dataDir, specs)
			}
			return runCompose(ctx, args, specs)
		},
	}
}

// runAttached runs the specs against a network that is already up.
//
// One function for all three ways in: which network, and where it came from, is
// the start state. This tool had only two of the three until that state existed
// — a spec declaring env.attach could not be run through it at all.
func runAttached(ctx context.Context, args map[string]any, start app.Start, chain string, rpcURLs []string, dataDir string, specs [][]byte) (string, error) {
	artifactRoot := argString(args, "artifactRoot", "")
	if artifactRoot == "" {
		var err error
		artifactRoot, err = os.MkdirTemp("", "cb-run")
		if err != nil {
			return "", fmt.Errorf("chainbench_run: temp dir: %w", err)
		}
	}
	root, err := invokeFeature[app.AttachRunIn, string](ctx, app.Deps{}.WithLogins(argString(args, "serverLogins", "")), "test.attach", attachInput(args, start, chain, rpcURLs, dataDir, artifactRoot, specs))
	if err != nil {
		return "", err
	}
	return formatRunSummary(root)
}

func attachInput(args map[string]any, start app.Start, chain string, rpcURLs []string, dataDir, artifactRoot string, specs [][]byte) app.AttachRunIn {
	return app.AttachRunIn{
		At: start.At, Declared: start.Declared,
		Chain: chain, RPCURLs: rpcURLs, DataDir: dataDir,
		ArtifactRoot: artifactRoot, Specs: specs,
		KeysDir: argString(args, "keysDir", ""), ReadOnlyKeys: argBool(args, "readOnlyKeys", false), NoNodeControl: argBool(args, "noNodeControl", false),
	}
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
	in, err := composeInput(args, specs, dataDir)
	if err != nil {
		return "", err
	}
	if argBool(args, "plan", false) {
		var plans []app.SuitePlan
		if argBool(args, "suite", false) {
			layout, plan, err := app.PlanSuiteLayout(ctx, in)
			if err != nil {
				return "", err
			}
			plans = []app.SuitePlan{{Spec: "suite", Layout: layout, Plan: plan}}
		} else {
			var err error
			plans, err = app.PlanSuites(ctx, in)
			if err != nil {
				return "", err
			}
		}
		raw, err := json.Marshal(plans)
		return string(raw), err
	}
	deps := app.Deps{}.WithLogins(argString(args, "serverLogins", ""))
	if len(specs) > 1 && !argBool(args, "suite", false) {
		out, err := invokeFeature[app.RunSuiteIn, app.RunSuitesOut](ctx, deps, "test.run-sequence", in)
		if err != nil {
			return "", err
		}
		raw, err := json.Marshal(out)
		if err != nil {
			return "", err
		}
		if out.Failed() {
			return string(raw), fmt.Errorf("sequential test run failed: %s", raw)
		}
		return string(raw), nil
	}
	out, err := invokeFeature[app.RunSuiteIn, app.RunSuiteOut](ctx, deps, "test.run", in)
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

// composeInput binds the same accepted inputs that the CLI passes to the engine.
func composeInput(args map[string]any, specs [][]byte, dataDir string) (app.RunSuiteIn, error) {
	in := app.RunSuiteIn{
		SpecContent: specs,
		DataDir:     dataDir,
		Chain:       argString(args, "chain", ""),
		Binary:      argString(args, "binary", ""),
		BPCount:     argInt(args, "bp", 0),
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
	}
	in.ReadOnlyKeys = argBool(args, "readOnlyKeys", false)
	in.ArtifactRoot = argString(args, "artifactRoot", "")
	in.WorkspaceConfigPath = argString(args, "workspaceConfig", "")
	if raw, ok := args["binaryFor"]; ok {
		b, err := json.Marshal(raw)
		if err != nil {
			return in, err
		}
		if err = json.Unmarshal(b, &in.BinaryOverrides); err != nil {
			return in, fmt.Errorf("binaryFor must map names to executable paths: %w", err)
		}
		for name, path := range in.BinaryOverrides {
			if name == "" || path == "" {
				return in, fmt.Errorf("binaryFor needs nonempty names and paths")
			}
		}
	}
	return in, nil
}
