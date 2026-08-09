package testspec

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// fakeNodeControl records the stop/start calls a fault action makes.
type fakeNodeControl struct {
	mu      sync.Mutex
	stopped []int
	started []int
	err     error
}

func (c *fakeNodeControl) Stop(_ context.Context, n node.Node) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	c.stopped = append(c.stopped, n.Index)
	return nil
}

func (c *fakeNodeControl) Start(_ context.Context, n node.Node) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.err != nil {
		return c.err
	}
	c.started = append(c.started, n.Index)
	return nil
}

// envWithNodes builds an environment of n validator nodes pointed at url.
func envWithNodes(t *testing.T, n int, url string) session.Environment {
	t.Helper()
	sess, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC(), nil)
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	env, err := sess.NewEnvironment("bbbbbbbbbbbb0000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	nodes := make([]node.Node, 0, n)
	for i := 1; i <= n; i++ {
		nodes = append(nodes, node.Node{Index: i, Role: node.RoleValidator, Host: "127.0.0.1", RPCURL: url})
	}
	env.PopulateNodeTable(node.NodeSet{Nodes: nodes})
	return env
}

func faultDeps(ctrl NodeControl) Deps {
	return Deps{
		RPC:     func(u string) *rpc.Client { return rpc.Dial(u) },
		Actions: NewRegistry(true),
		Nodes:   ctrl,
	}
}

func TestStopNodeAction_StopsTheSelectedNode(t *testing.T) {
	ctrl := &fakeNodeControl{}
	d := faultDeps(ctrl)
	env := envWithNodes(t, 4, "http://unused")

	act, ok := d.Actions.Action(actionStopNode)
	if !ok {
		t.Fatal("stopNode not registered")
	}
	if err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{"on": "bp2"}}); err != nil {
		t.Fatalf("stopNode: %v", err)
	}
	if len(ctrl.stopped) != 1 || ctrl.stopped[0] != 2 {
		t.Fatalf("stopped = %v, want [2]", ctrl.stopped)
	}
}

func TestStartNodeAction_StartsTheSelectedNode(t *testing.T) {
	ctrl := &fakeNodeControl{}
	d := faultDeps(ctrl)
	env := envWithNodes(t, 4, "http://unused")

	act, _ := d.Actions.Action(actionStartNode)
	if err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{"on": "bp3"}}); err != nil {
		t.Fatalf("startNode: %v", err)
	}
	if len(ctrl.started) != 1 || ctrl.started[0] != 3 {
		t.Fatalf("started = %v, want [3]", ctrl.started)
	}
}

func TestRestartNodeAction_StopsThenStarts(t *testing.T) {
	ctrl := &fakeNodeControl{}
	d := faultDeps(ctrl)
	env := envWithNodes(t, 4, "http://unused")

	act, _ := d.Actions.Action(actionRestartNode)
	if err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{"on": "bp1"}}); err != nil {
		t.Fatalf("restartNode: %v", err)
	}
	if len(ctrl.stopped) != 1 || ctrl.stopped[0] != 1 {
		t.Fatalf("stopped = %v, want [1]", ctrl.stopped)
	}
	if len(ctrl.started) != 1 || ctrl.started[0] != 1 {
		t.Fatalf("started = %v, want [1]", ctrl.started)
	}
}

func TestStopNodeAction_WithoutNodeControlIsAClearError(t *testing.T) {
	d := faultDeps(nil) // attach mode: no process control
	env := envWithNodes(t, 2, "http://unused")
	act, _ := d.Actions.Action(actionStopNode)
	err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{"on": "bp1"}})
	if err == nil {
		t.Fatal("expected an error when no node control is wired")
	}
	if !strings.Contains(err.Error(), "node control") {
		t.Fatalf("error should name the missing capability, got: %v", err)
	}
}

// peerRPC serves admin_nodeInfo per node and records admin_addPeer /
// admin_removePeer calls, so a partition can be checked without a real network.
type peerRPC struct {
	mu      sync.Mutex
	removed []string
	added   []string
}

func (p *peerRPC) server(t *testing.T, enode string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		_ = json.Unmarshal(body, &req)
		var result any
		switch req.Method {
		case "admin_nodeInfo":
			result = map[string]any{"enode": enode}
		case "admin_removePeer":
			p.mu.Lock()
			p.removed = append(p.removed, req.Params[0].(string))
			p.mu.Unlock()
			result = true
		case "admin_addPeer":
			p.mu.Lock()
			p.added = append(p.added, req.Params[0].(string))
			p.mu.Unlock()
			result = true
		default:
			http.Error(w, "unknown method "+req.Method, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

// envFromURLs builds an environment whose node i points at urls[i-1].
func envFromURLs(t *testing.T, urls []string) session.Environment {
	t.Helper()
	sess, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC(), nil)
	if err != nil {
		t.Fatalf("session.New: %v", err)
	}
	env, err := sess.NewEnvironment("cccccccccccc0000")
	if err != nil {
		t.Fatalf("NewEnvironment: %v", err)
	}
	nodes := make([]node.Node, 0, len(urls))
	for i, u := range urls {
		nodes = append(nodes, node.Node{Index: i + 1, Role: node.RoleValidator, Host: "127.0.0.1", RPCURL: u})
	}
	env.PopulateNodeTable(node.NodeSet{Nodes: nodes})
	return env
}

func TestPartitionAction_SeversEveryCrossGroupLink(t *testing.T) {
	rec := &peerRPC{}
	urls := make([]string, 4)
	for i := range urls {
		urls[i] = rec.server(t, "enode://node"+string(rune('1'+i))+"@127.0.0.1:30300").URL
	}
	d := faultDeps(nil) // partition needs RPC only, not process control
	env := envFromURLs(t, urls)

	act, ok := d.Actions.Action(actionPartition)
	if !ok {
		t.Fatal("partition not registered")
	}
	err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{
		"groups": []any{
			[]any{"bp1", "bp2"},
			[]any{"bp3", "bp4"},
		},
	}})
	if err != nil {
		t.Fatalf("partition: %v", err)
	}
	// 2x2 cross-group pairs, severed from both sides: 8 removePeer calls.
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.removed) != 8 {
		t.Fatalf("removePeer calls = %d, want 8: %v", len(rec.removed), rec.removed)
	}
	if len(rec.added) != 0 {
		t.Fatalf("partition should not add peers, got %v", rec.added)
	}
}

func TestPartitionAction_RequiresTwoGroups(t *testing.T) {
	d := faultDeps(nil)
	env := envWithNodes(t, 4, "http://unused")
	act, _ := d.Actions.Action(actionPartition)
	err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{
		"groups": []any{[]any{"bp1", "bp2"}},
	}})
	if err == nil {
		t.Fatal("expected an error for a single group")
	}
}

func TestHealPartitionAction_ReconnectsEveryPair(t *testing.T) {
	rec := &peerRPC{}
	urls := make([]string, 3)
	for i := range urls {
		urls[i] = rec.server(t, "enode://node"+string(rune('1'+i))+"@127.0.0.1:30300").URL
	}
	d := faultDeps(nil)
	env := envFromURLs(t, urls)

	act, ok := d.Actions.Action(actionHealPartition)
	if !ok {
		t.Fatal("healPartition not registered")
	}
	if err := act.Do(context.Background(), &ActionCtx{Env: env, Deps: &d, Args: map[string]any{}}); err != nil {
		t.Fatalf("healPartition: %v", err)
	}
	rec.mu.Lock()
	defer rec.mu.Unlock()
	// Full mesh over 3 nodes, both directions: 3*2 = 6 addPeer calls.
	if len(rec.added) != 6 {
		t.Fatalf("addPeer calls = %d, want 6: %v", len(rec.added), rec.added)
	}
}
