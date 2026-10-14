package testengine_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/testengine"
	"github.com/gorilla/websocket"
)

func TestExternalEndpointProfileAuthenticatesAllReadTransports(t *testing.T) {
	t.Setenv("CHAINBENCH_TEST_ENDPOINT_TOKEN", "test-token-only")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token-only" {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/metrics":
			_, _ = w.Write([]byte("profile_metric 42\n"))
			return
		case "/ws":
			conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			var request map[string]any
			if err := conn.ReadJSON(&request); err != nil {
				t.Error(err)
				return
			}
			_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "id": request["id"], "result": "0x1"})
			_ = conn.WriteJSON(map[string]any{"jsonrpc": "2.0", "method": "eth_subscription", "params": map[string]any{"subscription": "0x1", "result": map[string]any{"number": "0x2"}}})
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}
		var request struct {
			ID     any    `json:"id"`
			Method string `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": "0x2"})
	}))
	defer server.Close()
	profiles := []node.RPCEndpoint{{RPCURL: server.URL, Role: node.RoleBP, Auth: &node.EndpointAuth{Type: "jwt", Env: "CHAINBENCH_TEST_ENDPOINT_TOKEN"}}, {RPCURL: server.URL + "/en", Role: node.RoleEN, WSURL: "ws" + strings.TrimPrefix(server.URL, "http") + "/ws", MetricsURL: server.URL + "/metrics", Auth: &node.EndpointAuth{Type: "jwt", Env: "CHAINBENCH_TEST_ENDPOINT_TOKEN"}}}
	ns, err := node.AttachedSet("stablenet", "external", profiles)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: "stablenet", Nodes: ns, ArtifactRoot: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	spec := []byte(`{"schemaVersion":"2","kind":"case","id":"profile-read","requires":["rpc"],"chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"external","chain":"stablenet","attach":{"rpc":["http://unused"]}},"steps":[{"expect":"blockNumber","on":"en1","is":"2"},{"expect":"metric","on":"en1","name":"profile_metric","is":42},{"expect":"wsSubscribe","on":"en1","event":"newHeads","count":1,"timeout":"2s"}]}`)
	root, err := engine.Run(context.Background(), [][]byte{spec})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(root, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var saved struct{ Summary struct{ Pass, Fail int } }
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Summary.Pass != 1 || saved.Summary.Fail != 0 {
		t.Fatalf("profile reads failed: %s", raw)
	}
}
