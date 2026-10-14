package testhelper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestPartitionRemovesLateCrossGroupConnection(t *testing.T) {
	var mu sync.Mutex
	removed := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			ID     any
			Method string
		}
		_ = json.NewDecoder(r.Body).Decode(&q)
		mu.Lock()
		defer mu.Unlock()
		var result any = true
		switch q.Method {
		case "admin_nodeInfo":
			result = map[string]any{"enode": "enode://peer@127.0.0.1:30303"}
		case "admin_removePeer":
			removed++
		case "admin_peers":
			result = []any{}
			if removed == 2 {
				result = []any{map[string]any{"enode": "enode://peer@127.0.0.1:30303"}}
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
	}))
	defer srv.Close()
	env := envFromURLs(t, []string{srv.URL, srv.URL})
	d := faultDeps(nil)
	act, _ := d.Actions.Action(actionPartition)
	err := act.Do(context.Background(), &interp.ActionCtx{Deps: &d, Env: env, Args: map[string]any{"groups": []any{[]any{"node1"}, []any{"node2"}}}})
	mu.Lock()
	defer mu.Unlock()
	if err != nil || removed < 3 {
		t.Fatalf("late peer retained: removals=%d err=%v", removed, err)
	}
}

func TestPartitionObservationFailsClosed(t *testing.T) {
	for _, name := range []string{"null", "malformed", "rpc error"} {
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct{ ID any }
				_ = json.NewDecoder(r.Body).Decode(&q)
				reply := map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": nil}
				if name == "malformed" {
					reply["result"] = []any{map[string]any{"enode": "broken"}}
				}
				if name == "rpc error" {
					delete(reply, "result")
					reply["error"] = map[string]any{"code": -32000, "message": "unavailable"}
				}
				_ = json.NewEncoder(w).Encode(reply)
			}))
			defer srv.Close()
			env := envFromURLs(t, []string{srv.URL, srv.URL})
			d := faultDeps(nil)
			nodes := env.Nodes()
			err := settlePeerPartition(context.Background(), &interp.ActionCtx{Deps: &d, Env: env}, [][]node.Node{{nodes[0]}, {nodes[1]}}, map[int]string{1: "enode://first@127.0.0.1:30303", 2: "enode://second@127.0.0.1:30304"})
			if err == nil {
				t.Fatal("unverifiable partition accepted")
			}
		})
	}
}
