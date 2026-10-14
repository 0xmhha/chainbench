package testengine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

func TestConsensusGateUsesRecordedSuccessorChain(t *testing.T) {
	for _, chain := range []string{"wbft", "unknown-chain"} {
		t.Run(chain, func(t *testing.T) {
			requests := make(chan string, 8)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ID     any
					Method string
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				requests <- request.Method
				response := map[string]any{"jsonrpc": "2.0", "id": request.ID}
				if request.Method == "istanbul_getValidators" {
					response["result"] = []string{"0xvalidator"}
				} else {
					response["error"] = map[string]any{"code": -32601, "message": "method not found"}
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			s, err := session.New(t.TempDir(), "mixed-chain-proof", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			env, err := s.NewEnvironment("mixed")
			if err != nil {
				t.Fatal(err)
			}
			encoded, _ := json.Marshal(map[string]any{"index": 1, "role": "bp", "rpc_url": server.URL, "chain": chain})
			var n node.Node
			if err := json.Unmarshal(encoded, &n); err != nil {
				t.Fatal(err)
			}
			env.PopulateNodeTable(node.NodeSet{Chain: "wemix", Nodes: []node.Node{n}})
			rec := s.Test(1, "mixed")
			status, reason := consensusCapabilityGate("wemix", rpc.Dial)(t.Context(), dsl.Spec{Requires: []string{"consensus"}}, env, rec)
			var methods []string
			for len(requests) > 0 {
				methods = append(methods, <-requests)
			}
			if chain == "wbft" {
				if status != "" || len(methods) != 1 || methods[0] != "istanbul_getValidators" {
					t.Fatalf("wrong consensus dialect: status=%s reason=%s methods=%v", status, reason, methods)
				}
			} else if status != session.StatusBlocked || len(methods) != 0 {
				t.Fatalf("unknown recorded chain used a fallback: %s %s %v", status, reason, methods)
			}
			if err := s.Save(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(rec.Dir(), "observations", "capability-probes.json"))
			if err != nil {
				t.Fatal(err)
			}
			var observation struct{ Nodes []struct{ Chain string } }
			if err := json.Unmarshal(data, &observation); err != nil || len(observation.Nodes) != 1 || observation.Nodes[0].Chain != chain {
				t.Fatalf("per-node chain provenance absent: %s %v", data, err)
			}
		})
	}
}
