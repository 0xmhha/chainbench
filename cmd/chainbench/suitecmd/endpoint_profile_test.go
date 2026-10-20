package suitecmd_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/mcp"
)

func TestCLIMCPEndpointProfilesExpandAndSelectRolesIdentically(t *testing.T) {
	t.Setenv("CHAINBENCH_TEST_PROFILE_TOKEN", "test-token-only")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		result := any("0x63")
		if request.Method == "web3_clientVersion" {
			result = "Gstable/v-test"
		}
		if r.URL.Path == "/en" {
			if r.Header.Get("Authorization") != "Bearer test-token-only" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			result = "0x2a"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}))
	defer server.Close()
	t.Setenv("CHAINBENCH_TEST_PROFILE_RPC", server.URL)
	raw := []byte(`{"schemaVersion":"2","kind":"case","id":"external-profile","requires":["rpc"],"chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"external","chain":"stablenet","attach":{"endpoints":[{"rpc":"${CHAINBENCH_TEST_PROFILE_RPC}/bp","role":"bp"},{"rpc":"${CHAINBENCH_TEST_PROFILE_RPC}/en","role":"en","auth":{"type":"jwt","env":"CHAINBENCH_TEST_PROFILE_TOKEN"}}]}},"steps":[{"expect":"blockNumber","on":"en1","is":"42"}]}`)
	path := filepath.Join(t.TempDir(), "profile.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	output, _, err := runSplit(t, "run", "--json", "--artifact-root", t.TempDir(), path)
	if err != nil {
		t.Fatal(err, output)
	}
	var cli struct {
		Session string `json:"session"`
	}
	if err := json.Unmarshal([]byte(output), &cli); err != nil {
		t.Fatal(err)
	}
	requireProfilePass(t, cli.Session)
	mcpArtifacts := t.TempDir()
	request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"spec": string(raw), "artifactRoot": mcpArtifacts}}})
	if err != nil {
		t.Fatal(err)
	}
	var response struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		}
		Error any
	}
	if err := json.Unmarshal(mcp.Default("profile", "test").Handle(context.Background(), request), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != nil || response.Result.IsError || len(response.Result.Content) != 1 {
		t.Fatalf("MCP refused profile: %+v", response)
	}
	ids, err := session.List(mcpArtifacts)
	if err != nil || len(ids) != 1 {
		t.Fatal(err, ids)
	}
	requireProfilePass(t, filepath.Join(mcpArtifacts, ids[0]))
}

func requireProfilePass(t *testing.T, root string) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Summary struct{ Pass, Fail, Skip int }
	}
	if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatal(err)
	}
	if report.Summary.Pass != 1 || report.Summary.Fail != 0 || report.Summary.Skip != 0 {
		t.Fatalf("profile result: %s", raw)
	}
}
