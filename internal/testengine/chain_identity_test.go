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

func TestDeclaredChainIdentityGatesEveryEndpointBeforeHooks(t *testing.T) {
	for _, c := range []struct {
		name        string
		want        any
		replies     []any
		status      string
		httpFailure bool
	}{
		{"match", "1", []any{"0x1", "0x1"}, "pass", false},
		{"later node mismatch", "1", []any{"0x1", "0x2"}, "fail", false},
		{"first node mismatch", "1", []any{"0x2", "0x1"}, "fail", false},
		{"zero is explicit", 0, []any{"0x0", "0x0"}, "pass", false},
		{"zero mismatch", 0, []any{"0x0", "0x1"}, "fail", false},
		{"mismatch outranks unknown", "1", []any{"0x2", nil}, "fail", false},
		{"null response", "1", []any{"0x1", nil}, "blocked", false},
		{"decimal protocol response", "291", []any{"0x123", "123"}, "blocked", false},
		{"transport failure", "1", []any{"0x1", "0x1"}, "blocked", true},
	} {
		t.Run(c.name, func(t *testing.T) {
			var identity, steps, preSpec atomic.Int32
			urls := []string{}
			for index, reply := range c.replies {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req struct {
						ID     any
						Method string
					}
					_ = json.NewDecoder(r.Body).Decode(&req)
					if req.Method == "eth_chainId" {
						identity.Add(1)
						if c.httpFailure && index == 1 {
							http.Error(w, "credentials must not be echoed", http.StatusServiceUnavailable)
							return
						}
						_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": reply})
						return
					}
					steps.Add(1)
					_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x10"})
				}))
				defer server.Close()
				urls = append(urls, server.URL)
			}
			eng, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", RPCURLs: urls, ArtifactRoot: t.TempDir(), PreSpec: func(context.Context, session.Environment) error { preSpec.Add(1); return nil }})
			if err != nil {
				t.Fatal(err)
			}
			raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "identity", "chainPreset": map[string]any{"chain": "stablenet", "values": map[string]any{"chainId": c.want}}, "hooks": map[string]any{"pre": []any{map[string]any{"do": "read", "source": "blockNumber", "save": "height"}}}, "steps": []any{map[string]any{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": 1}}})
			root, err := eng.Run(context.Background(), [][]byte{raw})
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "tests", "001_identity")
			data, err := os.ReadFile(filepath.Join(dir, "status.json"))
			if err != nil {
				t.Fatal(err)
			}
			var result struct{ Result, Reason string }
			_ = json.Unmarshal(data, &result)
			if result.Result != c.status {
				t.Errorf("identity gate: %s", data)
			}
			if identity.Load() != 2 {
				t.Errorf("did not probe every endpoint: %d", identity.Load())
			}
			if c.status == "pass" {
				if steps.Load() == 0 || preSpec.Load() != 1 {
					t.Errorf("matching nodes did not execute: steps=%d pre=%d", steps.Load(), preSpec.Load())
				}
			} else if steps.Load() != 0 || preSpec.Load() != 0 {
				t.Errorf("wrong/unknown network reached hooks: steps=%d pre=%d", steps.Load(), preSpec.Load())
			}
			probe, err := os.ReadFile(filepath.Join(dir, "observations", "chain-identity.json"))
			if err != nil {
				t.Error("identity observation missing", err)
			} else if len(probe) == 0 {
				t.Error("empty observation")
			}
		})
	}
}

func TestChainIdentityGateUsesEachRecordedNodesAuthentication(t *testing.T) {
	t.Setenv("IDENTITY_NODE_ONE", "identity-one-only")
	t.Setenv("IDENTITY_NODE_TWO", "identity-two-only")
	var calls atomic.Int32
	nodes := node.NodeSet{}
	for i, token := range []string{"identity-one-only", "identity-two-only"} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+token {
				http.Error(w, "denied", http.StatusUnauthorized)
				return
			}
			var q struct {
				ID     any
				Method string
			}
			_ = json.NewDecoder(r.Body).Decode(&q)
			if q.Method == "eth_chainId" {
				calls.Add(1)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": "0x1"})
		}))
		defer server.Close()
		nodes.Nodes = append(nodes.Nodes, node.Node{Index: i + 1, Role: node.RoleEN, RPCURL: server.URL, Auth: node.Auth{"type": "jwt", "env": []string{"IDENTITY_NODE_ONE", "IDENTITY_NODE_TWO"}[i]}})
	}
	eng, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", Nodes: nodes, ArtifactRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte(`{"schemaVersion":"2","kind":"case","id":"auth-identity","chainPreset":{"chain":"stablenet","values":{"chainId":"1"}},"steps":[{"expect":"blockNumber","on":"node2","is":1}]}`)
	root, err := eng.Run(context.Background(), [][]byte{raw})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "tests", "001_auth-identity", "status.json"))
	if err != nil {
		t.Fatal(err)
	}
	var status struct{ Result string }
	_ = json.Unmarshal(content, &status)
	if status.Result != "pass" || calls.Load() != 2 {
		t.Fatalf("identity ignored recorded node auth: %s calls=%d", content, calls.Load())
	}
}
