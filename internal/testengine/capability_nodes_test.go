package testengine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestConsensusCapabilityChecksEveryAuthenticatedNodeBeforeExecution(t *testing.T) {
	for _, c := range []struct {
		name, status string
		replies      []string
	}{
		{"all supported", "pass", []string{"supported", "supported"}},
		{"later method missing", "skip", []string{"supported", "unsupported"}},
		{"later endpoint unavailable", "blocked", []string{"supported", "unknown"}},
		{"first method missing still probes later", "skip", []string{"unsupported", "supported"}},
		{"unknown outranks unsupported", "blocked", []string{"unsupported", "unknown"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			var probes, steps, pre atomic.Int32
			nodes := node.NodeSet{}
			for i, reply := range c.replies {
				token, variable := []string{"first-token", "second-token"}[i], []string{"CAP_NODE_FIRST", "CAP_NODE_SECOND"}[i]
				t.Setenv(variable, token)
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Header.Get("Authorization") != "Bearer "+token {
						http.Error(w, "denied", http.StatusUnauthorized)
						return
					}
					var request struct {
						ID     any
						Method string
					}
					if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
						t.Error(err)
						return
					}
					if request.Method == "istanbul_getValidators" {
						probes.Add(1)
						switch reply {
						case "unknown":
							http.Error(w, "private upstream detail", http.StatusServiceUnavailable)
						case "unsupported":
							_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "error": map[string]any{"code": -32601, "message": "method not found"}})
						default:
							_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": []string{"0xpublic-validator"}})
						}
						return
					}
					steps.Add(1)
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": "0x10"})
				}))
				defer server.Close()
				nodes.Nodes = append(nodes.Nodes, node.Node{Index: i + 1, Role: node.RoleEN, RPCURL: server.URL, Auth: node.Auth{"type": "jwt", "env": variable}})
			}
			eng, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", Nodes: nodes, Caps: []string{"consensus"}, ArtifactRoot: t.TempDir(), PreSpec: func(context.Context, session.Environment) error { pre.Add(1); return nil }})
			if err != nil {
				t.Fatal(err)
			}
			raw := []byte(`{"schemaVersion":"2","kind":"case","id":"all-node-consensus","requires":["rpc","consensus"],"chainPreset":{"chain":"stablenet"},"hooks":{"pre":[{"do":"read","source":"blockNumber","save":"height"}]},"steps":[{"expect":"blockNumber","on":"node2","is":16}]}`)
			root, err := eng.Run(context.Background(), [][]byte{raw})
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "tests", "001_all-node-consensus")
			data, err := os.ReadFile(filepath.Join(dir, "status.json"))
			if err != nil {
				t.Fatal(err)
			}
			var verdict struct{ Result string }
			_ = json.Unmarshal(data, &verdict)
			if verdict.Result != c.status || probes.Load() != 2 {
				t.Errorf("incomplete consensus gate: %s probes=%d", data, probes.Load())
			}
			if c.status != "pass" && (steps.Load() != 0 || pre.Load() != 0) {
				t.Errorf("inadequate nodes executed pre/steps: %d/%d", pre.Load(), steps.Load())
			}
			data, err = os.ReadFile(filepath.Join(dir, "observations", "capability-probes.json"))
			if err != nil {
				t.Fatal(err)
			}
			var observation struct {
				Nodes []struct {
					Node           int
					Status, Source string
				}
			}
			if err := json.Unmarshal(data, &observation); err != nil {
				t.Fatal(err)
			}
			if len(observation.Nodes) != 2 {
				t.Fatalf("per-node provenance absent: %s", data)
			}
			for i, n := range observation.Nodes {
				if n.Node != i+1 || n.Status != c.replies[i] || n.Source == "" {
					t.Fatalf("incorrect node observation: %s", data)
				}
			}
		})
	}
}
