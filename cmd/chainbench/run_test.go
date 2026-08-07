package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockRPCNode serves canned JSON-RPC results keyed by method.
func mockRPCNode(t *testing.T, results map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		res := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if v, ok := results[req.Method]; ok {
			res["result"] = v
		}
		_ = json.NewEncoder(w).Encode(res)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func writeSpec(t *testing.T, spec map[string]any) string {
	t.Helper()
	b, _ := json.Marshal(spec)
	p := filepath.Join(t.TempDir(), "spec.json")
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRunCmd_AttachMockRPC(t *testing.T) {
	srv := mockRPCNode(t, map[string]any{
		"eth_chainId":     "0x539", // 1337
		"eth_blockNumber": "0x10",  // 16
	})
	specPath := writeSpec(t, map[string]any{
		"schemaVersion": "1",
		"id":            "cli-smoke",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{
			{"assert": "chainId", "expected": 1337},
			{"assert": "blockNumber", "expected": 1},
		},
	})

	out, err := run(t, "run", "--chain", "stablenet", "--rpc", srv.URL, "--artifact-root", t.TempDir(), specPath)
	if err != nil {
		t.Fatalf("run: %v\n%s", err, out)
	}
	if !strings.Contains(out, "pass=1") {
		t.Fatalf("expected pass=1 in output:\n%s", out)
	}
	if !strings.Contains(out, "cli-smoke") {
		t.Fatalf("expected the test id in output:\n%s", out)
	}
}

func TestRunCmd_FailingSpecExitsNonZero(t *testing.T) {
	srv := mockRPCNode(t, map[string]any{"eth_chainId": "0x1"}) // chainId 1
	specPath := writeSpec(t, map[string]any{
		"schemaVersion": "1",
		"id":            "cli-fail",
		"chain":         map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions":    []map[string]any{{"assert": "chainId", "expected": 999}},
	})
	out, err := run(t, "run", "--chain", "stablenet", "--rpc", srv.URL, "--artifact-root", t.TempDir(), specPath)
	if err == nil {
		t.Fatalf("expected a non-nil error for a failing spec:\n%s", out)
	}
	if !strings.Contains(out, "fail=1") {
		t.Fatalf("expected fail=1 in output:\n%s", out)
	}
}

func TestRunCmd_RequiresMode(t *testing.T) {
	specPath := writeSpec(t, map[string]any{
		"schemaVersion": "1", "id": "x",
		"chain":      map[string]any{"name": "stablenet", "binary": "go-stablenet"},
		"assertions": []map[string]any{{"assert": "True"}},
	})
	// Neither --rpc nor --binary given.
	if _, err := run(t, "run", "--chain", "stablenet", specPath); err == nil {
		t.Fatal("expected error when neither attach nor local mode is selected")
	}
	// No --chain.
	if _, err := run(t, "run", "--rpc", "http://127.0.0.1:1", specPath); err == nil {
		t.Fatal("expected error when --chain is missing")
	}
}
