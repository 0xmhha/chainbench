package testhelper

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// chainNode is a fake node: a head height and a hash per block number. It
// records which blocks it was asked for.
type chainNode struct {
	head   uint64
	hashes map[string]string // "0x5" -> hash
	mu     sync.Mutex
	asked  []string
}

func (n *chainNode) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		_ = json.Unmarshal(body, &req)
		var res any
		switch req.Method {
		case "eth_blockNumber":
			res = hexOf(n.head)
		case "eth_getBlockByNumber":
			tag, _ := req.Params[0].(string)
			n.mu.Lock()
			n.asked = append(n.asked, tag)
			n.mu.Unlock()
			h, ok := n.hashes[tag]
			if !ok {
				res = nil
				break
			}
			res = map[string]any{"number": tag, "hash": h, "parentHash": "0x00", "timestamp": "0x1", "transactions": []any{}}
		default:
			http.Error(w, "unknown method "+req.Method, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": res})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func hexOf(n uint64) string {
	const digits = "0123456789abcdef"
	if n == 0 {
		return "0x0"
	}
	var b []byte
	for ; n > 0; n /= 16 {
		b = append([]byte{digits[n%16]}, b...)
	}
	return "0x" + string(b)
}

// envOf is a session environment whose node table holds the given URLs.
func envOf(t *testing.T, urls ...string) session.Environment {
	t.Helper()
	sess, err := session.New(t.TempDir(), "test", time.Unix(0, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	env, err := sess.NewEnvironment("bbbbbbbbbbbb0000")
	if err != nil {
		t.Fatal(err)
	}
	nodes := make([]node.Node, 0, len(urls))
	for i, u := range urls {
		nodes = append(nodes, node.Node{Index: i + 1, Role: node.RoleBP, Host: "127.0.0.1", RPCURL: u})
	}
	env.PopulateNodeTable(node.NodeSet{Nodes: nodes})
	return env
}

func checkSameBlockHash(t *testing.T, env session.Environment, on []node.Node, spec map[string]any) session.AssertResult {
	t.Helper()
	d := deps()
	spec["assert"] = assertSameBlockHash
	r, err := (sameBlockHashAssertion{}).Check(context.Background(), &interp.AssertCtx{Deps: &d, Env: env, On: on, Spec: spec})
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return r
}

// TestSameBlockHash_WithoutOnComparesEveryNode: a node that disagrees fails the
// check even when the case names no nodes. It used to compare node1 with
// itself and pass.
func TestSameBlockHash_WithoutOnComparesEveryNode(t *testing.T) {
	a := &chainNode{head: 5, hashes: map[string]string{"0x5": "0xaaaa"}}
	b := &chainNode{head: 5, hashes: map[string]string{"0x5": "0xbbbb"}}
	env := envOf(t, a.serve(t).URL, b.serve(t).URL)
	if r := checkSameBlockHash(t, env, nil, map[string]any{}); r.Pass {
		t.Fatalf("two nodes with different hashes passed: %+v", r.Actual)
	}
	b.hashes["0x5"] = "0xaaaa"
	if r := checkSameBlockHash(t, env, nil, map[string]any{}); !r.Pass {
		t.Fatalf("two nodes that agree failed: %v", r.Source)
	}
}

// TestSameBlockHash_LatestIsTheLowestCommonHead: nodes a block apart are
// compared at the lower head, which both hold, not at their own latest.
func TestSameBlockHash_LatestIsTheLowestCommonHead(t *testing.T) {
	a := &chainNode{head: 7, hashes: map[string]string{"0x6": "0xcc", "0x7": "0xnew"}}
	b := &chainNode{head: 6, hashes: map[string]string{"0x6": "0xcc"}}
	env := envOf(t, a.serve(t).URL, b.serve(t).URL)
	r := checkSameBlockHash(t, env, nil, map[string]any{"block": "latest"})
	if !r.Pass {
		t.Fatalf("nodes that agree on their common block 6 failed: %v", r.Source)
	}
	if len(a.asked) == 0 || a.asked[len(a.asked)-1] != "0x6" {
		t.Errorf("the higher node was asked for %v, want block 0x6", a.asked)
	}
}

// TestSameBlockHash_OnEachStillNamesTheNodes: an explicit onEach compares only
// the nodes it names.
func TestSameBlockHash_OnEachStillNamesTheNodes(t *testing.T) {
	a := &chainNode{head: 3, hashes: map[string]string{"0x3": "0xdd"}}
	b := &chainNode{head: 3, hashes: map[string]string{"0x3": "0xdd"}}
	c := &chainNode{head: 3, hashes: map[string]string{"0x3": "0xee"}} // not named
	ua, ub, uc := a.serve(t).URL, b.serve(t).URL, c.serve(t).URL
	env := envOf(t, ua, ub, uc)
	on := []node.Node{{Index: 1, RPCURL: ua}, {Index: 2, RPCURL: ub}}
	if r := checkSameBlockHash(t, env, on, map[string]any{}); !r.Pass {
		t.Fatalf("the two named nodes agree, yet: %v", r.Source)
	}
}
