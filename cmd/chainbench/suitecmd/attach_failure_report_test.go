package suitecmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestAttachedFailedCasePreservesJSONSessionReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&request)
		var result any = "0x2"
		if request.Method == "web3_clientVersion" {
			result = "Gstable/v-test"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
	}))
	defer server.Close()
	raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "attached-failure", "requires": []string{"rpc"}, "chainPreset": map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "external", "chain": "stablenet", "attach": map[string]any{"endpoints": []any{map[string]any{"rpc": server.URL, "role": "en"}}}}, "steps": []any{map[string]any{"expect": "blockNumber", "on": "en1", "is": "1", "compare": "Equal"}}})
	path := filepath.Join(t.TempDir(), "case.json")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	output, progress, err := runSplit(t, "run", "--json", "--artifact-root", t.TempDir(), path)
	if codeOf(t, err) != 1 {
		t.Fatalf("assertion failure did not return code 1: %v %s", err, progress)
	}
	var report struct {
		Session string                                  `json:"session"`
		Summary struct{ Pass, Fail, Blocked, Skip int } `json:"summary"`
	}
	if parseErr := json.Unmarshal([]byte(output), &report); parseErr != nil {
		t.Fatalf("failed attached run lost its JSON report: %v stdout=%q error=%v", parseErr, output, err)
	}
	if report.Session == "" || report.Summary.Fail != 1 || report.Summary.Pass != 0 || report.Summary.Blocked != 0 || report.Summary.Skip != 0 {
		t.Fatalf("failed run report is incomplete: %s", output)
	}
	if _, err := os.Stat(filepath.Join(report.Session, "session.json")); err != nil {
		t.Fatal(err)
	}
}
