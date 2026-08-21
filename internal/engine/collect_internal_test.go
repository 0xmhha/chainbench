package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// mockRPCServer serves canned JSON-RPC results keyed by method.
func mockRPCServer(t *testing.T, results map[string]any) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if v, ok := results[req.Method]; ok {
			resp["result"] = v
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func collectEnv(t *testing.T, nodes ...node.Node) session.Environment {
	t.Helper()
	s, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	env, err := s.NewEnvironment("ffffffffffff0000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	env.PopulateNodeTable(node.NodeSet{Nodes: nodes})
	return env
}

// TestStartCollection_MirrorsChainstateAndLogs proves live collection publishes
// both a chainstate snapshot and tailed log lines to the bus.
func TestStartCollection_MirrorsChainstateAndLogs(t *testing.T) {
	env := collectEnv(t, node.Node{Index: 1, Role: node.RoleValidator, RPCURL: "http://n1"})
	if err := os.WriteFile(env.LogPath("node1"), []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	probe := func(context.Context, string) (collector.NodeState, error) {
		return collector.NodeState{Height: 7, Peers: 2, HeadHash: "0xh", HeadMiner: "0xA"}, nil
	}
	bus := obs.NewBus()
	sub := bus.Subscribe()

	stop := startCollection(context.Background(), env, bus, probe, 5*time.Millisecond)

	var sawChainstate, sawLog bool
	deadline := time.After(2 * time.Second)
	for !sawChainstate || !sawLog {
		select {
		case e := <-sub:
			switch {
			case e.Message == "chainstate":
				if h, ok := e.Fields["heights"].(map[string]uint64); ok && h["node1"] == 7 {
					sawChainstate = true
				}
			case e.Message == "hello world" && e.Fields["log"] == true:
				sawLog = true
			}
		case <-deadline:
			t.Fatalf("timeout: chainstate=%v log=%v", sawChainstate, sawLog)
		}
	}

	if err := stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	bus.Close()

	// Chainstate samples are also persisted to the session as jsonl.
	recs := readChainstate(t, filepath.Join(env.ChainstateDir(), chainstateFile))
	if len(recs) == 0 {
		t.Fatal("expected chainstate.jsonl to have samples")
	}
	if recs[len(recs)-1].Heights["node1"] != 7 {
		t.Fatalf("last persisted sample = %+v, want node1 height 7", recs[len(recs)-1])
	}
}

// TestRPCProbe_ReadsNodeState checks the RPC-backed probe maps client reads into
// a NodeState.
func TestRPCProbe_ReadsNodeState(t *testing.T) {
	// A nil dialer defaults to rpc.Dial; here we inject a dialer against a mock.
	// Reuse the mock RPC used elsewhere by dialing a test server.
	srv := mockRPCServer(t, map[string]any{
		"eth_blockNumber": "0x9", // 9
		"net_peerCount":   "0x2",
		"eth_getBlockByNumber": map[string]any{
			"number": "0x9", "hash": "0xhead", "miner": "0xMINER",
		},
	})
	probe := rpcProbe(nil)
	st, err := probe(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("probe: %v", err)
	}
	if st.Height != 9 || st.Peers != 2 || st.HeadHash != "0xhead" || st.HeadMiner != "0xMINER" {
		t.Fatalf("probe state = %+v", st)
	}
}
