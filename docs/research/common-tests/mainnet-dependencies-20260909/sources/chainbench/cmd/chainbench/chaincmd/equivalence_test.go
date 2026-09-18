package chaincmd_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/mcp"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// U2 routed the chain group through app, which is what makes these possible:
// both surfaces now call the same entry point, so a difference between them is
// a defect rather than the expected consequence of two implementations
// (worklist §1l).
//
// What is compared is the workspace each surface produced, not the words each
// printed. Composing is a side effect on disk — that state is the answer, and a
// CLI writes a table about it for a person while MCP writes JSON for a program.

// callMCP invokes a tool and returns its text content.
func callMCP(t *testing.T, tool string, args map[string]any) string {
	t.Helper()
	req, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": args},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw := mcp.Default("chainbench", "test").Handle(context.Background(), req)
	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("%s: bad response: %v (%s)", tool, err, raw)
	}
	if resp.Error != nil {
		t.Fatalf("%s: %s", tool, resp.Error.Message)
	}
	if len(resp.Result.Content) == 0 {
		t.Fatalf("%s: no content in %s", tool, raw)
	}
	if resp.Result.IsError {
		t.Fatalf("%s failed: %s", tool, resp.Result.Content[0].Text)
	}
	return resp.Result.Content[0].Text
}

// workspaceState reads a composed workspace's recorded state as a value.
//
// It names workspace.json rather than looking for "a JSON file": the directory
// also holds process.json, which is {"procs": []} for a freshly composed
// network. An earlier version of this walked the directory and compared that
// one instead, so it passed by comparing two empty documents and missed a
// deliberately injected divergence.
func workspaceState(t *testing.T, dir string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, "workspace.json"))
	if err != nil {
		t.Fatalf("no workspace.json under %s: the surface composed nothing (%v)", dir, err)
	}
	var v map[string]any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("workspace.json is not JSON: %v", err)
	}
	// Two runs land in two directories and happen at two instants, so the
	// workspace path and the step timestamps are what must differ. Everything
	// else — the chain, the keys, the target, which steps ran and what each
	// reported — is the composition, and that must not.
	normalize(v, dir)
	return v
}

// normalize replaces the run-specific parts of a state document: the workspace
// path wherever it appears in a string, and the timestamps, which are dropped.
func normalize(v any, dir string) {
	switch t := v.(type) {
	case map[string]any:
		delete(t, "at")
		for k, sub := range t {
			if str, ok := sub.(string); ok {
				t[k] = strings.ReplaceAll(str, dir, "<workspace>")
				continue
			}
			normalize(sub, dir)
		}
	case []any:
		for i, sub := range t {
			if str, ok := sub.(string); ok {
				t[i] = strings.ReplaceAll(str, dir, "<workspace>")
				continue
			}
			normalize(sub, dir)
		}
	}
}

// TestEquivalence_ChainNewComposesTheSameWorkspace: `chain new` and
// chainbench_chain_new are the same use case behind two surfaces, so given the
// same inputs they must leave the same workspace behind.
//
// Before U2 the CLI called chainsetup directly while the tool went through app.
// Nothing checked that the two agreed, and either could have picked up a
// different default without the other noticing.
func TestEquivalence_ChainNewComposesTheSameWorkspace(t *testing.T) {
	preset := filepath.Join("..", "..", "..", "keys", "preset")
	cliDir, mcpDir := t.TempDir(), t.TempDir()

	out, err := run(t, "chain", "new", "--workspace-dir", cliDir, "--chain", "stablenet", "--keys", preset)
	if err != nil {
		t.Fatalf("CLI chain new: %v\n%s", err, out)
	}
	callMCP(t, "chainbench_chain_new", map[string]any{
		"workspaceDir": mcpDir, "chain": "stablenet", "keys": preset,
	})

	cli, mcpState := workspaceState(t, cliDir), workspaceState(t, mcpDir)
	// A guard with teeth: the document has to carry the composition, not just
	// exist. Comparing two empty files is how the first version of this test
	// passed while missing an injected divergence.
	if steps, ok := cli["steps"].(map[string]any); !ok || len(steps) == 0 || cli["chain"] != "stablenet" {
		t.Fatalf("the CLI recorded no composition, so agreeing about it proves nothing: %#v", cli)
	}
	if !reflect.DeepEqual(cli, mcpState) {
		t.Errorf("the two surfaces composed different workspaces.\n  CLI: %#v\n  MCP: %#v", cli, mcpState)
	}
}

// TestEquivalence_ChainStatusReportsTheSameComposition: both surfaces read one
// workspace, so the facts they report about it must be the same. The renderings
// differ and are allowed to — MCP answers with the state as JSON, the CLI lays
// the same state out for a person.
//
// The comparison reads the tool's JSON and requires the CLI's text to carry
// what it names. Checking for a few hand-picked words instead is not enough: an
// earlier version looked for "stablenet" and passed even when the tool stopped
// reporting the chain, because the word also occurs inside a step's detail
// line.
func TestEquivalence_ChainStatusReportsTheSameComposition(t *testing.T) {
	preset := filepath.Join("..", "..", "..", "keys", "preset")
	dir := t.TempDir()
	if out, err := run(t, "chain", "new", "--workspace-dir", dir, "--chain", "stablenet", "--keys", preset); err != nil {
		t.Fatalf("chain new: %v\n%s", err, out)
	}

	cli, err := run(t, "chain", "status", "--workspace-dir", dir)
	if err != nil {
		t.Fatalf("CLI chain status: %v\n%s", err, cli)
	}

	var state struct {
		Chain string                    `json:"chain"`
		Steps map[string]map[string]any `json:"steps"`
	}
	raw := callMCP(t, "chainbench_chain_status", map[string]any{"workspaceDir": dir})
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		t.Fatalf("the tool's answer is not the state as JSON: %v\n%s", err, raw)
	}
	if state.Chain == "" || len(state.Steps) == 0 {
		t.Fatalf("the tool reported no composition, so agreeing about it proves nothing:\n%s", raw)
	}

	// The chain the tool names has to be the chain the CLI names, and on the
	// line that says so rather than anywhere in the output.
	if !strings.Contains(cli, "chain: "+state.Chain) {
		t.Errorf("the tool reports chain %q but the CLI does not say so:\n%s", state.Chain, cli)
	}
	// Every step the tool reports as run has to appear in the CLI's listing.
	for name := range state.Steps {
		if !strings.Contains(cli, name) {
			t.Errorf("the tool reports step %q as recorded; the CLI does not list it:\n%s", name, cli)
		}
	}
}

// TestEquivalence_TheWorkspaceDefaultIsOneDefault: with no workspace named,
// the default comes from app rather than from each surface's own arithmetic.
// Two surfaces computing their own is how one composes somewhere the other
// cannot find, and it was possible until U2 moved the default into app.
//
// `chain new` is the verb that takes the default (status and the later steps
// act on a workspace the operator names), and it prints the path it chose
// before using it, precisely so this is observable.
func TestEquivalence_TheWorkspaceDefaultIsOneDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // the same variable's name on Windows

	preset := filepath.Join("..", "..", "..", "keys", "preset")
	out, err := run(t, "chain", "new", "--chain", "stablenet", "--keys", preset)
	if err != nil {
		t.Fatalf("chain new with no workspace: %v\n%s", err, out)
	}
	if !strings.Contains(out, home) {
		t.Fatalf("the default workspace is not under this test's home, so the surface computed its own:\n%s", out)
	}

	// And it is the path app would have chosen, not one this surface invented.
	want, err := app.DefaultWorkspaceDir(app.Deps{})
	if err != nil {
		t.Fatal(err)
	}
	// The timestamp segment moves between the two calls, so the root is what
	// can be compared; that is where the two surfaces would diverge.
	root := filepath.Dir(filepath.Dir(want))
	if !strings.Contains(out, root) {
		t.Errorf("the surface's default is not under app's root %s:\n%s", root, out)
	}
}
