package network_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/network" // register the cases

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// mockNode answers eth_getBlockByNumber(0) with a fixed genesis hash and
// net_peerCount with 2, so a two-node set agrees on genesis and has peers.
func mockNode(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int   `json:"id"`
			Params []any `json:"params"`
		}
		_ = json.Unmarshal(body, &req)
		var result any
		switch {
		case strings.Contains(string(body), "eth_blockNumber"):
			result = "0x64" // head = 100
		case strings.Contains(string(body), "eth_getBlockByNumber"):
			// echo the requested block as its number + timestamp so
			// block-progression sees strictly-increasing values; a fixed hash
			// so genesis-hash-agreement sees agreement.
			blk := "0x0"
			if len(req.Params) > 0 {
				if s, ok := req.Params[0].(string); ok {
					blk = s
				}
			}
			result = map[string]any{"number": blk, "hash": "0xabc123", "timestamp": blk}
		case strings.Contains(string(body), "net_peerCount"):
			result = "0x2"
		case strings.Contains(string(body), "admin_peers"):
			result = []map[string]any{{"id": "abc123", "name": "geth"}}
		default:
			result = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runCase(t *testing.T, name string, endpoints int) testkit.Result {
	t.Helper()
	srv := mockNode(t)
	eps := make([]node.RPCEndpoint, endpoints)
	for i := range eps {
		eps[i] = node.RPCEndpoint{RPCURL: srv.URL}
	}
	ns, _ := node.AttachedSet("wbft", "local", eps)
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{name}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 {
		t.Fatalf("%s: want 1 result, got %+v", name, rep.Results)
	}
	return rep.Results[0]
}

func TestNetworkCases_Registered(t *testing.T) {
	want := map[string]bool{"genesis-hash-agreement": false, "peers-connected": false, "admin-peers-populated": false}
	for _, c := range testkit.Cases() {
		if _, ok := want[c.Name]; ok {
			want[c.Name] = true
			if c.Category != "network" {
				t.Errorf("%s category = %q, want network", c.Name, c.Category)
			}
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("case %q not registered", name)
		}
	}
}

func TestGenesisHashAgreement_Passes(t *testing.T) {
	if r := runCase(t, "genesis-hash-agreement", 3); r.Status != testkit.StatusPass {
		t.Fatalf("genesis-hash-agreement: %+v", r)
	}
}

func TestBlockProgression_Passes(t *testing.T) {
	if r := runCase(t, "block-progression", 1); r.Status != testkit.StatusPass {
		t.Fatalf("block-progression: %+v", r)
	}
}

func TestPeersConnected_Passes(t *testing.T) {
	if r := runCase(t, "peers-connected", 2); r.Status != testkit.StatusPass {
		t.Fatalf("peers-connected: %+v", r)
	}
}

func TestAdminPeersPopulated_Passes(t *testing.T) {
	if r := runCase(t, "admin-peers-populated", 2); r.Status != testkit.StatusPass {
		t.Fatalf("admin-peers-populated: %+v", r)
	}
}
