package suitecmd_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/report"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/mcp"
)

// TestNativeRequirementsRunSurfacesAndEvidence executes the public command
// handlers and MCP protocol against real nodes; no harness component is mocked.
func TestNativeRequirementsRunSurfacesAndEvidence(t *testing.T) {
	binary := os.Getenv("GSTABLE_BIN")
	if binary == "" {
		t.Skip("native requirement proof requires GSTABLE_BIN")
	}
	root := os.Getenv("WEBUI_RUNTIME_ROOT")
	if root == "" {
		root = t.TempDir()
	}
	fixture, err := os.MkdirTemp(root, "requirements-native-")
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(fixture, "w")
	artifacts := filepath.Join(fixture, "reports")
	t.Cleanup(func() {
		if _, err := app.ChainStop(context.Background(), app.Deps{}, app.ChainStopIn{DataDir: workspace}); err != nil {
			t.Errorf("stop owned native fixture: %v", err)
			return
		}
		if _, err := app.ChainRm(context.Background(), app.Deps{}, app.ChainRmIn{DataDir: workspace}); err != nil {
			t.Errorf("reclaim owned native data: %v", err)
		}
	})
	var paths, raws []string
	for i, id := range []string{"first-pass", "middle-fails", "last-pass"} {
		expected := 8283
		if i == 1 {
			expected = 1
		}
		raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": id, "requires": []string{"rpc"}, "chainPreset": map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "native", "chain": "stablenet", "binaries": map[string]any{"default": "gstable"}, "topology": map[string]any{"bp": 4}}, "steps": []map[string]any{{"expect": "chainId", "compare": "Equal", "is": expected}}})
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(fixture, id+".json")
		if err := os.WriteFile(path, raw, 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
		raws = append(raws, string(raw))
	}
	base := []string{"run", "--workspace-dir", workspace, "--artifact-root", artifacts, "--keys", repoKeys(t), "--read-only-keys", "--binary", binary, "--wait-blocks", "1", "--json", "--no-skips"}
	stdout, stderr, runErr := runSplit(t, append(append([]string{}, base...), paths...)...)
	if codeOf(t, runErr) != 1 {
		t.Fatalf("real sequence exit=%d\n%s\n%s", codeOf(t, runErr), stdout, stderr)
	}
	var sequence struct {
		Runs []struct {
			Spec    string `json:"spec"`
			Session string `json:"session"`
			Error   string `json:"error"`
			app.RunSummary
		} `json:"runs"`
	}
	if err := json.Unmarshal([]byte(stdout), &sequence); err != nil {
		t.Fatal(err, stdout)
	}
	if len(sequence.Runs) != 3 {
		t.Fatal("native sequence dropped a definition")
	}
	seen := map[string]bool{}
	for i, run := range sequence.Runs {
		if run.Spec != paths[i] || run.Error != "" || len(run.Tests) != 1 {
			t.Fatalf("definition did not execute: %+v", run)
		}
		want := "pass"
		if i == 1 {
			want = "fail"
		}
		if run.Tests[0].Status != want || run.Summary.Skip != 0 || run.Summary.Blocked != 0 {
			t.Fatal(run)
		}
		if seen[run.Session] {
			t.Fatal("native definitions share an artifact root")
		}
		seen[run.Session] = true
		rep, err := report.Read(run.Session)
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Tests) != 1 || rep.Tests[0].Status != want || len(rep.Tests[0].Evidence) < 3 {
			t.Fatal("root report detached from executed definition")
		}
		for _, link := range rep.Tests[0].Evidence {
			if _, err := os.Stat(filepath.Join(run.Session, link)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(fixture, "sequence.json"), []byte(stdout), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "sequence.log"), []byte(stderr), 0o600); err != nil {
		t.Fatal(err)
	}
	// The same DSL executed singly still returns the single-run document and 0.
	singleOut, singleLog, singleErr := runSplit(t, append(append([]string{}, base...), paths[2])...)
	if singleErr != nil {
		t.Fatal(singleErr, singleLog, singleOut)
	}
	var single struct {
		Session string `json:"session"`
		app.RunSummary
	}
	if err := json.Unmarshal([]byte(singleOut), &single); err != nil || single.Summary.Pass != 1 || single.Summary.Skip != 0 {
		t.Fatal(err, singleOut)
	}
	if err := os.WriteFile(filepath.Join(fixture, "single.json"), []byte(singleOut), 0o600); err != nil {
		t.Fatal(err)
	}
	// One MCP call consumes the same DSL and produces real per-test evidence.
	mcpArtifacts := filepath.Join(fixture, "mcp-reports")
	request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"dataDir": workspace, "binary": binary, "keysDir": repoKeys(t), "readOnlyKeys": true, "artifactRoot": mcpArtifacts, "specs": []string{raws[0], raws[2]}, "suite": true, "waitBlocks": 1}}})
	if err != nil {
		t.Fatal(err)
	}
	response := mcp.Default("requirements", "native").Handle(context.Background(), request)
	var decoded struct {
		Result struct {
			IsError bool `json:"isError"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(response, &decoded); err != nil || decoded.Error != nil || decoded.Result.IsError {
		t.Fatalf("native MCP run refused: %v %s", err, response)
	}
	sessions, err := session.List(mcpArtifacts)
	if err != nil || len(sessions) != 1 {
		t.Fatal(err, sessions)
	}
	rep, err := report.Read(filepath.Join(mcpArtifacts, sessions[0]))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Pass != 2 || rep.Summary.Fail != 0 || rep.Summary.Skip != 0 || len(rep.Tests) != 2 {
		t.Fatal(rep)
	}
	if err := os.WriteFile(filepath.Join(fixture, "mcp.json"), response, 0o600); err != nil {
		t.Fatal(err)
	}
	sourceHashes := map[string]string{}
	for _, pattern := range []string{"*.go", "../../../internal/core/session/*.go", "../../../internal/testengine/*.go", "../../../internal/app/runsuites.go", "../../../internal/mcp/run_tool.go"} {
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			content, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			sha := sha256.Sum256(content)
			sourceHashes[file] = hex.EncodeToString(sha[:])
		}
	}
	content, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	sha := sha256.Sum256(content)
	proof := map[string]any{"binary": binary, "binarySHA256": hex.EncodeToString(sha[:]), "sourceFiles": sourceHashes, "sequenceSessions": seen, "singleSession": single.Session, "mcpSession": sessions[0], "sequenceVerdicts": []string{"pass", "fail", "pass"}, "mcpPassed": 2}
	encoded, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(fixture, "proof.json")
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Logf("native requirement proof: %s", path)
}
