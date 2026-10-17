package testengine

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

func TestConsensusGateSelectsDeclaredUpgradePhase(t *testing.T) {
	for _, tc := range []struct {
		name, height, style, failure, recorded, outer, target string
		upgrade, requires                                     bool
		wantStatus                                            session.TestStatus
		wantMethods                                           []string
		wantConsensus, wantHeight                             string
	}{
		{name: "before fork", height: "0x77", upgrade: true, requires: true, wantMethods: []string{"eth_blockNumber", "eth_wemixInfo", "eth_call", "eth_call"}, wantConsensus: "wemix", wantHeight: "119"},
		{name: "at fork", height: "0x78", upgrade: true, requires: true, wantMethods: []string{"eth_blockNumber", "istanbul_getValidators"}, wantConsensus: "wbft", wantHeight: "120"},
		{name: "after fork", height: "0x79", upgrade: true, requires: true, wantMethods: []string{"eth_blockNumber", "istanbul_getValidators"}, wantConsensus: "wbft", wantHeight: "121"},
		{name: "default concurrent style", height: "0x77", style: "default", upgrade: true, requires: true, wantMethods: []string{"eth_blockNumber", "eth_wemixInfo", "eth_call", "eth_call"}, wantConsensus: "wemix", wantHeight: "119"},
		{name: "no declaration never falls back", height: "0x77", requires: true, wantStatus: session.StatusBlocked, wantMethods: []string{"istanbul_getValidators"}},
		{name: "restart never assumes concurrent engine", height: "0x77", style: dsl.UpgradeRestart, upgrade: true, requires: true, wantStatus: session.StatusBlocked, wantMethods: []string{"istanbul_getValidators"}},
		{name: "other recorded chain never inherits source", height: "0x77", recorded: "stablenet", upgrade: true, requires: true, wantStatus: session.StatusBlocked, wantMethods: []string{"istanbul_getValidators"}},
		{name: "successor without phase reader is blocked", height: "0x77", recorded: "stablenet", target: "stablenet", upgrade: true, requires: true, wantStatus: session.StatusBlocked, wantMethods: []string{"eth_blockNumber"}},
		{name: "same consensus family keeps its method", height: "0x79", outer: "stablenet", upgrade: true, requires: true, wantMethods: []string{"istanbul_getValidators"}},
		{name: "head RPC failure prevents consensus query", height: "0x77", failure: "head", upgrade: true, requires: true, wantStatus: session.StatusBlocked, wantMethods: []string{"eth_blockNumber"}},
		{name: "malformed head prevents consensus query", height: "0xzz", upgrade: true, requires: true, wantStatus: session.StatusBlocked, wantMethods: []string{"eth_blockNumber"}},
		{name: "pre fork error never tries successor", height: "0x77", failure: "poa", upgrade: true, requires: true, wantStatus: session.StatusBlocked, wantMethods: []string{"eth_blockNumber", "eth_wemixInfo"}, wantConsensus: "wemix", wantHeight: "119"},
		{name: "post fork error never tries predecessor", height: "0x78", failure: "wbft", upgrade: true, requires: true, wantStatus: session.StatusBlocked, wantMethods: []string{"eth_blockNumber", "istanbul_getValidators"}, wantConsensus: "wbft", wantHeight: "120"},
		{name: "not required makes no calls", height: "0x77", upgrade: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var mu sync.Mutex
			var methods []string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ID     any
					Method string
					Params []json.RawMessage
				}
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					return
				}
				mu.Lock()
				methods = append(methods, request.Method)
				mu.Unlock()
				response := map[string]any{"jsonrpc": "2.0", "id": request.ID}
				fail := request.Method == "eth_blockNumber" && tc.failure == "head" || request.Method == "eth_wemixInfo" && tc.failure == "poa" || request.Method == "istanbul_getValidators" && (tc.failure == "wbft" || tc.height == "0x77" || tc.height == "0xzz")
				if fail {
					response["error"] = map[string]any{"code": -32000, "message": "private-upstream-error"}
				} else {
					switch request.Method {
					case "eth_blockNumber":
						response["result"] = tc.height
					case "eth_wemixInfo":
						response["result"] = map[string]string{"governance": "0x0000000000000000000000000000000000000010"}
					case "eth_call":
						response["result"] = fmt.Sprintf("0x%064x", 1)
					case "istanbul_getValidators":
						response["result"] = []string{"0x0000000000000000000000000000000000000001"}
					default:
						t.Errorf("unexpected RPC method %s", request.Method)
					}
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			outer, recorded := tc.outer, tc.recorded
			if outer == "" {
				outer = "wemix"
			}
			if recorded == "" {
				recorded = "wbft"
			}
			target := tc.target
			if target == "" {
				target = "wbft"
			}
			spec := dsl.Spec{Chain: dsl.ChainSpec{Name: outer, BinaryChains: map[string]string{"next": target}}}
			if tc.requires {
				spec.Requires = []string{"consensus"}
			}
			if tc.upgrade {
				at := int64(120)
				style := tc.style
				switch style {
				case "":
					style = dsl.UpgradeConcurrent
				case "default":
					style = ""
				}
				spec.EnvUpgrade = &dsl.UpgradeV2{Fork: "croissant", At: &at, From: "default", To: "next", Style: style}
			}
			s, err := session.New(t.TempDir(), "upgrade-phase", time.Now())
			if err != nil {
				t.Fatal(err)
			}
			env, err := s.NewEnvironment("mixed")
			if err != nil {
				t.Fatal(err)
			}
			env.PopulateNodeTable(node.NodeSet{Chain: outer, Nodes: []node.Node{{Index: 1, Chain: recorded, Role: node.RoleEN, RPCURL: server.URL}}})
			record := s.Test(1, "phase")
			status, reason := consensusCapabilityGate(outer, rpc.Dial)(t.Context(), spec, env, record)
			if status != tc.wantStatus {
				t.Errorf("verdict %s (%s), want %s", status, reason, tc.wantStatus)
			}
			mu.Lock()
			if !reflect.DeepEqual(methods, tc.wantMethods) {
				t.Errorf("RPC dialect %v, want %v", methods, tc.wantMethods)
			}
			mu.Unlock()
			if !tc.requires {
				return
			}
			if err := s.Save(); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(record.Dir(), "observations", "capability-probes.json"))
			if err != nil {
				t.Fatal(err)
			}
			var observed struct{ Nodes []map[string]any }
			if err := json.Unmarshal(data, &observed); err != nil || len(observed.Nodes) != 1 {
				t.Fatalf("missing per-node evidence: %s %v", data, err)
			}
			n := observed.Nodes[0]
			if strings.Contains(string(data), "private-upstream-error") {
				t.Errorf("upstream text leaked into evidence: %s", data)
			}
			if n["chain"] != recorded {
				t.Errorf("recorded binary identity changed: %s", data)
			}
			if tc.wantConsensus != "" && (n["consensusChain"] != tc.wantConsensus || n["height"] != tc.wantHeight || n["forkBlock"] != float64(120)) {
				t.Errorf("active consensus and observed boundary missing: %s", data)
			}
			if tc.failure == "head" && (n["source"] != "eth_blockNumber" || n["failureKind"] != "rpc_error" || n["rpcErrorCode"] != float64(-32000)) {
				t.Errorf("head failure provenance lost: %s", data)
			}
		})
	}
}
