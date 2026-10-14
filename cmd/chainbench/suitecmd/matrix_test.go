package suitecmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/mcp"
)

func matrixFixture(t *testing.T) (string, []string, []string) {
	t.Helper()
	dir := t.TempDir()
	presets := []string{}
	for index, head := range []string{"0x3", "0x4"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				ID     any
				Method string
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			result := head
			if req.Method == "web3_clientVersion" {
				result = "Gstable/v-test"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
		}))
		t.Cleanup(server.Close)
		id := []string{"matrix-a", "matrix-b"}[index]
		raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": id, "chain": "stablenet", "attach": map[string]any{"rpc": []string{server.URL}}})
		path := filepath.Join(dir, id+".json")
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		presets = append(presets, path)
	}
	paths, contents := []string{}, []string{}
	for _, id := range []string{"first", "second"} {
		raw := `{"schemaVersion":"2","kind":"case","id":"` + id + `","requires":["rpc"],"chainPreset":"matrix-a","steps":[{"expect":"blockNumber","is":"3","compare":"Equal"}]}`
		path := filepath.Join(dir, id+".json")
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
		contents = append(contents, raw)
	}
	return dir, presets, append(paths, contents...)
}

func TestMatrixPreflightsEveryTargetBeforeCallingAnyEndpoint(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
	defer server.Close()
	dir, presets, inputs := matrixFixture(t)
	raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "preflight", "chain": "stablenet", "attach": map[string]any{"rpc": []string{server.URL}}})
	if err := os.WriteFile(presets[0], raw, 0600); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runSplit(t, "run", "--json", "--artifact-root", filepath.Join(dir, "artifacts"), "--chain-presets", presets[0], "--chain-presets", filepath.Join(dir, "missing.json"), inputs[0])
	if err == nil || !strings.Contains(err.Error(), "missing.json") || stdout != "" || calls.Load() != 0 {
		t.Fatalf("preflight mutated the first target: %v output=%s calls=%d", err, stdout, calls.Load())
	}
	if _, err := os.Stat(filepath.Join(dir, "artifacts")); !os.IsNotExist(err) {
		t.Fatal("preflight created artifacts", err)
	}
}

func TestMatrixInfrastructureFailureDoesNotHideTheLaterTarget(t *testing.T) {
	_, presets, inputs := matrixFixture(t)
	// Missing transport credentials fail before a test body; an unreachable RPC
	// alone is intentionally recorded by the existing assertion as test failure.
	t.Setenv("CHAINBENCH_MATRIX_MISSING_AUTH", "")
	raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "unreachable", "chain": "stablenet", "attach": map[string]any{"endpoints": []any{map[string]any{"rpc": "http://127.0.0.1:1", "auth": map[string]any{"type": "jwt", "env": "CHAINBENCH_MATRIX_MISSING_AUTH"}}}}})
	if err := os.WriteFile(presets[0], raw, 0600); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runSplit(t, "run", "--json", "--artifact-root", t.TempDir(), "--chain-presets", presets[0], "--chain-presets", presets[1], inputs[0])
	if codeOf(t, err) != 2 {
		t.Fatal("infrastructure lost priority", err, stdout)
	}
	var doc struct {
		Runs []struct {
			PresetID, Session, Error string
			Attempted                bool
		}
		SetupErrors int
		Summary     struct{ Fail int }
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Runs) != 2 || doc.SetupErrors != 1 || doc.Summary.Fail != 1 || doc.Runs[0].Error == "" || doc.Runs[1].Session == "" || !doc.Runs[1].Attempted {
		t.Fatalf("later target lost: %s", stdout)
	}
}

func TestMCPMatrixSchemaAndMalformedInputFailClosed(t *testing.T) {
	dir, presets, inputs := matrixFixture(t)
	for _, bad := range []any{"matrix-a", []any{presets[0], true}, nil, []any{}} {
		request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"spec": inputs[2], "chainPresets": bad, "baseDir": dir, "artifactRoot": filepath.Join(dir, "invalid-artifacts")}}})
		response := mcp.Default("matrix", "test").Handle(t.Context(), request)
		var doc struct{ Result struct{ IsError bool } }
		if err := json.Unmarshal(response, &doc); err != nil || !doc.Result.IsError {
			t.Fatalf("bad targets accepted: %v %s", err, response)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "invalid-artifacts")); !os.IsNotExist(err) {
		t.Fatal("bad input created artifacts", err)
	}
	request := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	var doc struct {
		Result struct {
			Tools []struct {
				Name        string
				InputSchema struct {
					Properties map[string]struct {
						Type     string
						MinItems int
					}
				}
			}
		}
	}
	if err := json.Unmarshal(mcp.Default("matrix", "test").Handle(t.Context(), request), &doc); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tool := range doc.Result.Tools {
		if tool.Name == "chainbench_run" {
			field := tool.InputSchema.Properties["chainPresets"]
			found = field.Type == "array" && field.MinItems == 1
		}
	}
	if !found {
		t.Fatal("matrix input contract missing from MCP")
	}
}

func requireMatrixReport(t *testing.T, raw []byte) {
	t.Helper()
	var doc struct {
		Runs []struct {
			PresetID, Session string
			Tests             []struct{ PresetID, Status string }
			Summary           struct{ Pass, Fail, Blocked, Skip int }
		}
		Summary struct{ Pass, Fail, Blocked, Skip int }
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err, string(raw))
	}
	if len(doc.Runs) != 2 || doc.Summary.Pass != 2 || doc.Summary.Fail != 2 || doc.Summary.Blocked != 0 || doc.Summary.Skip != 0 {
		t.Fatalf("matrix results were not separated: %s", raw)
	}
	if doc.Runs[0].Session == doc.Runs[1].Session {
		t.Fatal("targets reused a session")
	}
	for index, row := range doc.Runs {
		want := []string{"matrix-a", "matrix-b"}[index]
		if row.PresetID != want || len(row.Tests) != 2 {
			t.Fatalf("wrong preset or cases: %+v", row)
		}
		for _, result := range row.Tests {
			if result.PresetID != want {
				t.Fatalf("summary lost preset: %+v", result)
			}
		}
		for _, name := range []string{"session.json", "report.json"} {
			content, err := os.ReadFile(filepath.Join(row.Session, name))
			if err != nil {
				t.Fatal(err)
			}
			var record struct{ Tests []struct{ PresetID string } }
			if err := json.Unmarshal(content, &record); err != nil {
				t.Fatal(err)
			}
			for _, result := range record.Tests {
				if result.PresetID != want {
					t.Fatalf("%s lost preset: %s", name, content)
				}
			}
		}
		statuses, err := filepath.Glob(filepath.Join(row.Session, "tests", "*", "status.json"))
		if err != nil || len(statuses) != 2 {
			t.Fatal(err, statuses)
		}
		for _, p := range statuses {
			content, _ := os.ReadFile(p)
			var record struct{ PresetID string }
			if err := json.Unmarshal(content, &record); err != nil || record.PresetID != want {
				t.Fatalf("status lost preset: %v %s", err, content)
			}
		}
	}
}

func TestCLIAndMCPRunTheSameSuiteOnEveryPreset(t *testing.T) {
	dir, presets, inputs := matrixFixture(t)
	args := []string{"run", "--json", "--artifact-root", t.TempDir()}
	for _, p := range presets {
		args = append(args, "--chain-presets", p)
	}
	stdout, _, err := runSplit(t, append(args, inputs[:2]...)...)
	if stdout == "" {
		t.Fatalf("matrix produced no report: %v", err)
	}
	if codeOf(t, err) != 1 {
		t.Fatalf("expected test failures, got %v %s", err, stdout)
	}
	requireMatrixReport(t, []byte(stdout))
	request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"specs": inputs[2:], "chainPresets": presets, "baseDir": dir, "artifactRoot": t.TempDir()}}})
	response := mcp.Default("matrix", "test").Handle(t.Context(), request)
	var envelope struct {
		Result struct {
			Content []struct{ Text string }
			IsError bool
		}
		Error any
	}
	if err := json.Unmarshal(response, &envelope); err != nil || envelope.Error != nil || !envelope.Result.IsError || len(envelope.Result.Content) != 1 {
		t.Fatalf("MCP matrix refusal: %v %s", err, response)
	}
	requireMatrixReport(t, []byte(envelope.Result.Content[0].Text))
}

func TestSingleRunSummaryRetainsTheDeclaredPresetID(t *testing.T) {
	_, _, inputs := matrixFixture(t)
	stdout, _, err := runSplit(t, "run", "--json", "--artifact-root", t.TempDir(), inputs[0])
	if err != nil {
		t.Fatal(err, stdout)
	}
	var doc struct{ Tests []struct{ PresetID string } }
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Tests) != 1 || doc.Tests[0].PresetID != "matrix-a" {
		t.Fatalf("summary lost the declared preset: %s", stdout)
	}
}
