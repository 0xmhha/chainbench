package testengine_test

import (
	"context"
	"encoding/json"
	"github.com/0xmhha/chainbench/internal/testhelper"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testspec"

	"github.com/0xmhha/chainbench/internal/testengine"
)

// mockRPC serves canned JSON-RPC results keyed by method.
func mockRPC(t *testing.T, results map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		res, ok := results[req.Method]
		if !ok {
			http.Error(w, "unknown method "+req.Method, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": res})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// runEnv builds a real session environment whose primary node targets url and
// returns a fresh test record to run against.
func runEnv(t *testing.T, url string) (session.Environment, session.TestRecord) {
	t.Helper()
	sess, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	env, err := sess.NewEnvironment("aaaaaaaaaaaa0000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	env.PopulateNodeTable(node.NodeSet{Nodes: []node.Node{
		{Index: 1, Role: node.RoleValidator, Host: "127.0.0.1", RPCURL: url},
	}})
	return env, sess.Test(1, "T1")
}

func specWithAssertions(t *testing.T, assertions []map[string]any) testspec.Spec {
	t.Helper()
	raw, _ := json.Marshal(map[string]any{
		"schemaVersion": "1",
		"id":            "T1",
		"chain":         map[string]any{"name": "wbft", "binary": "go-wbft"},
		"assertions":    assertions,
	})
	spec, err := testspec.Parse(raw)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return spec
}

func runSpecDeps() testspec.Deps {
	return testspec.Deps{
		RPC:     func(u string) *rpc.Client { return rpc.Dial(u) },
		Actions: testhelper.Registry(),
	}
}

// TestNewRunSpec_Vertical proves the full run vertical: spec -> interpreter ->
// built-in RPC assertions -> mock chain -> recorded status.
func TestNewRunSpec_Vertical(t *testing.T) {
	srv := mockRPC(t, map[string]any{
		"eth_chainId":     "0x539", // 1337
		"eth_blockNumber": "0x10",  // 16
	})
	run := testengine.NewRunSpec(runSpecDeps())

	cases := []struct {
		name       string
		assertions []map[string]any
		want       session.TestStatus
	}{
		{
			name: "all pass",
			assertions: []map[string]any{
				{"assert": "chainId", "expected": 1337},
				{"assert": "blockNumber", "expected": 1},
			},
			want: session.StatusPass,
		},
		{
			name:       "one assertion fails",
			assertions: []map[string]any{{"assert": "chainId", "expected": 999}},
			want:       session.StatusFail,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env, rec := runEnv(t, srv.URL)
			got, err := run(context.Background(), specWithAssertions(t, tc.assertions), env, rec)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			if got != tc.want {
				t.Fatalf("status = %q, want %q", got, tc.want)
			}
		})
	}
}
