package resourcecmd_test

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/0xmhha/chainbench/internal/mcp"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// This file is the worked example for the U track's gate (worklist §1l).
//
// The rule the track enforces is that every surface reaches a feature through
// app. The property that rule exists for is that the surfaces answer alike, and
// an import list is only a proxy for it — a CLI and an MCP tool can both import
// app and still be wired to different calls, or read different defaults, and no
// import check notices. So each feature that moves under app arrives with a
// test of this shape, and this is the shape.
//
// resource pool is used because both surfaces already go through app.NetPool,
// which makes this a test that passes today and would fail the moment the two
// drift. Three other pairs are in the same position (chain show, hardfork,
// resource plan); the remaining 22 verified pairs are not, and each of those is
// a U item.
//
// What is compared is the answer, not the rendering. A CLI prints a table for a
// person and JSON on request; MCP always returns JSON. Requiring the bytes to
// match would pin formatting instead of behaviour, so both answers are decoded
// and compared as values.

// mcpJSON calls an MCP tool and decodes the JSON document it returns.
func mcpJSON(t *testing.T, tool string, args map[string]any) any {
	t.Helper()
	s := mcp.Default("chainbench", "test")
	req, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": args},
	})
	if err != nil {
		t.Fatal(err)
	}
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
	raw := s.Handle(context.Background(), req)
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
	var v any
	if err := json.Unmarshal([]byte(resp.Result.Content[0].Text), &v); err != nil {
		t.Fatalf("%s: content is not JSON: %v (%s)", tool, err, resp.Result.Content[0].Text)
	}
	return v
}

// decode reads a CLI command's --json output as a value.
func decode(t *testing.T, out string) any {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatalf("CLI output is not JSON: %v (%s)", err, out)
	}
	return v
}

// TestParity_ResourcePool: the CLI and the MCP tool answer the same question,
// so they must give the same answer.
//
// Both are wired to app.NetPool today. If one is later pointed at the module
// directly, or picks up a different default, the two answers separate and this
// fails — which is the whole point, since nothing else would notice.
func TestParity_ResourcePool(t *testing.T) {
	// A home of this test's own, so the pool is read from an empty tree rather
	// than from whatever the developer's machine happens to hold. Both surfaces
	// see the same one, which is what makes the comparison meaningful.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // the same variable's name on Windows

	cli, err := run(t, "resource", "pool", "--json")
	if err != nil {
		t.Fatalf("CLI: %v\n%s", err, cli)
	}
	got, want := decode(t, cli), mcpJSON(t, "chainbench_resource_pool", map[string]any{})
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the two surfaces disagree about the pool.\n  CLI: %#v\n  MCP: %#v", got, want)
	}
	// A guard against the comparison passing because both said nothing.
	m, ok := got.(map[string]any)
	if !ok || len(m) == 0 {
		t.Fatalf("the answer is empty, so agreeing about it proves nothing: %#v", got)
	}
	if _, ok := os.LookupEnv("HOME"); !ok {
		t.Fatal("HOME was not set, so the test read the developer's real tree")
	}
}
