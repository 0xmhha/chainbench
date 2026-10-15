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

func TestContractCapabilityUsesEachRecordedChain(t *testing.T) {
	for _, tc := range []struct {
		name, outer, recorded, capability, address string
		blocked                                    bool
	}{
		{"recorded successor", "wemix", "wbft", "contract:govStaking", "0x0000000000000000000000000000000000001001", false},
		{"unknown recorded chain", "stablenet", "unknown-chain", "contract:govCouncil", "", true},
		{"old record falls back", "stablenet", "", "contract:govCouncil", "0x0000000000000000000000000000000000001004", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     any
					Method string
					Params []json.RawMessage
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				calls++
				var address string
				if len(req.Params) > 0 {
					_ = json.Unmarshal(req.Params[0], &address)
				}
				if req.Method != "eth_getCode" || address != tc.address {
					t.Errorf("wrong contract probe: %s %s", req.Method, address)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": "0x6000"})
			}))
			defer server.Close()
			s, err := session.New(t.TempDir(), "recorded-contract", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			env, err := s.NewEnvironment("mixed")
			if err != nil {
				t.Fatal(err)
			}
			env.PopulateNodeTable(node.NodeSet{Chain: tc.outer, Nodes: []node.Node{{Index: 1, Role: node.RoleBP, Chain: tc.recorded, RPCURL: server.URL}}})
			rec := s.Test(1, "contract")
			status, why := networkCapabilityGate(tc.outer, rpc.Dial, nil)(t.Context(), dsl.Spec{Requires: []string{tc.capability}}, env, rec)
			if tc.blocked {
				if status != session.StatusBlocked || calls != 0 {
					t.Fatalf("unknown recorded chain used fallback: %s %s calls=%d", status, why, calls)
				}
			} else if status != "" || calls != 1 {
				t.Fatalf("wrong contract table: %s %s calls=%d", status, why, calls)
			}
			if err := s.Save(); err != nil {
				t.Fatal(err)
			}
			b, err := os.ReadFile(filepath.Join(rec.Dir(), "observations/network-capability-probes.json"))
			if err != nil {
				t.Fatal(err)
			}
			var observation struct {
				Probes []struct {
					Nodes []struct{ Chain, Address, Reason string }
				}
			}
			if err := json.Unmarshal(b, &observation); err != nil {
				t.Fatal(err)
			}
			if len(observation.Probes) != 1 || len(observation.Probes[0].Nodes) != 1 {
				t.Fatal(string(b))
			}
			p := observation.Probes[0].Nodes[0]
			expected := tc.recorded
			if expected == "" {
				expected = tc.outer
			}
			if p.Chain != expected || p.Address != tc.address {
				t.Fatalf("missing recorded chain/address provenance: %s", b)
			}
			if tc.blocked && p.Reason != "recorded_chain_unavailable" {
				t.Fatal(string(b))
			}
		})
	}
}
