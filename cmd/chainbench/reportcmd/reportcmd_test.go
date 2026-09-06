package reportcmd_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/reportcmd"
	"github.com/0xmhha/chainbench/internal/mcp"
)

// report and log read what a run left behind. Both refuse without being told
// where to look, which is the behaviour these pin — until U1 moved them out of
// package main there was no way to check it.

func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(reportcmd.NewReport(), reportcmd.NewLog())
	var buf strings.Builder
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

// TestReport_RefusesWithoutAWorkspace: with no directory there is no run to
// report on, and an empty report would read as a run that produced nothing.
func TestReport_RefusesWithoutAWorkspace(t *testing.T) {
	out, err := run(t, "report")
	if err == nil {
		t.Fatalf("report with no workspace was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "--workspace-dir is required") {
		t.Errorf("the error does not name the missing flag: %v", err)
	}
}

// TestLog_RefusesWithoutADirectory: same reason, and the same distinction
// between "nothing matched" and "nowhere was searched".
func TestLog_RefusesWithoutADirectory(t *testing.T) {
	out, err := run(t, "log")
	if err == nil {
		t.Fatalf("log with no directory was accepted:\n%s", out)
	}
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("the error does not say what is missing: %v", err)
	}
}

// TestReport_OnAnEmptyDirectorySaysSo: a directory with no run in it is a fact,
// not a crash.
func TestReport_OnAnEmptyDirectorySaysSo(t *testing.T) {
	out, err := run(t, "report", "--workspace-dir", t.TempDir())
	if err == nil && strings.TrimSpace(out) == "" {
		t.Fatal("an empty workspace produced neither an error nor a word of output")
	}
}

// U5 routed report and log through app, so both surfaces now read the same
// session and the same log files (worklist §1l).

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
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("%s: bad response: %v (%s)", tool, err, raw)
	}
	if len(resp.Result.Content) == 0 || resp.Result.IsError {
		t.Fatalf("%s failed: %s", tool, raw)
	}
	return resp.Result.Content[0].Text
}

// writeSession lays down a session with one report, which is what both
// surfaces read.
func writeSession(t *testing.T, dir string) {
	t.Helper()
	rep := map[string]any{
		"session":   "UTC-20260101-000000",
		"command":   "chainbench run spec.json",
		"startedAt": "2026-01-01T00:00:00Z",
		"summary":   map[string]any{"pass": 2, "fail": 1, "blocked": 0, "skip": 0},
		"tests": []map[string]any{
			{"seq": 1, "id": "first", "env": "stablenet", "status": "pass"},
			{"seq": 2, "id": "second", "env": "stablenet", "status": "fail"},
			{"seq": 3, "id": "third", "env": "stablenet", "status": "pass"},
		},
	}
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "report.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestParity_Report: one session, two surfaces reading it, so the verdicts they
// report must be the same.
//
// The layouts differ and are allowed to — the CLI draws a table for a person,
// MCP writes a line per test — so what is compared is the facts: every test's
// id and status, and the tally.
func TestParity_Report(t *testing.T) {
	dir := t.TempDir()
	writeSession(t, dir)

	cli, err := run(t, "report", "--workspace-dir", dir)
	if err != nil {
		t.Fatalf("CLI report: %v\n%s", err, cli)
	}
	mcpOut := callMCP(t, "chainbench_report", map[string]any{"workspaceDir": dir})

	for _, want := range []string{"first", "second", "third", "pass=2", "fail=1"} {
		inCLI, inMCP := strings.Contains(cli, want), strings.Contains(mcpOut, want)
		if inCLI != inMCP {
			t.Errorf("the surfaces disagree about %q: CLI=%v MCP=%v\n  CLI: %s\n  MCP: %s",
				want, inCLI, inMCP, cli, mcpOut)
		}
		if !inCLI {
			t.Errorf("neither surface reported %q, so agreeing proves nothing", want)
		}
	}
}

// TestParity_LogSearch: the same pattern over the same logs has to match the
// same lines. Reading logs used to be written twice, once per surface.
func TestParity_LogSearch(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	lines := "INFO [01-01|00:00:00.000] Starting peer\n" +
		"WARN [01-01|00:00:01.000] Dropping peer badpeer\n" +
		"ERROR [01-01|00:00:02.000] Sealing failed\n"
	if err := os.WriteFile(filepath.Join(dir, "logs", "node1.log"), []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}

	cli, err := run(t, "log", "--workspace-dir", dir, "--pattern", "peer")
	if err != nil {
		t.Fatalf("CLI log: %v\n%s", err, cli)
	}
	mcpOut := callMCP(t, "chainbench_log", map[string]any{"workspaceDir": dir, "pattern": "peer"})

	// Both must find the two peer lines and neither the sealing one.
	for _, want := range []string{"Starting peer", "Dropping peer"} {
		if !strings.Contains(cli, want) || !strings.Contains(mcpOut, want) {
			t.Errorf("a surface missed %q.\n  CLI: %s\n  MCP: %s", want, cli, mcpOut)
		}
	}
	if strings.Contains(cli, "Sealing failed") != strings.Contains(mcpOut, "Sealing failed") {
		t.Errorf("the surfaces disagree about a non-matching line.\n  CLI: %s\n  MCP: %s", cli, mcpOut)
	}
}
