package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestRPCHealthStrictOptionalReplies(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		block, peers, client any
		peerCode, clientCode int
		pass                 bool
	}{
		{"available", "0x1", "0x0", "Gstable/test", 0, 0, true},
		{"optional methods absent", "0x0", nil, nil, -32601, -32601, true},
		{"bad height", "hello", "0x1", "Gstable/test", 0, 0, false},
		{"null height", nil, "0x1", "Gstable/test", 0, 0, false},
		{"numeric height", 1, "0x1", "Gstable/test", 0, 0, false},
		{"bad peer count", "0x1", "bad", "Gstable/test", 0, 0, false},
		{"empty client", "0x1", "0x1", "", 0, 0, false},
		{"wrong client type", "0x1", "0x1", false, 0, 0, false},
		{"peer internal error", "0x1", nil, "Gstable/test", -32603, 0, false},
		{"client permission error", "0x1", "0x1", nil, 0, -32000, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			answers := map[string]rpcAnswer{"eth_blockNumber": {result: tc.block}, "net_peerCount": {result: tc.peers}, "web3_clientVersion": {result: tc.client}}
			if tc.peerCode != 0 {
				answers["net_peerCount"] = rpcAnswer{code: tc.peerCode, message: "probe failure"}
			}
			if tc.clientCode != 0 {
				answers["web3_clientVersion"] = rpcAnswer{code: tc.clientCode, message: "probe failure"}
			}
			srv := mockRPCAnswers(t, answers)
			d := deps()
			a, ok := d.Actions.Assertion("rpcHealth")
			if !ok {
				t.Fatal("missing public rpcHealth observation")
			}
			result, err := a.Check(t.Context(), &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}}, Spec: map[string]any{}})
			if result.Pass != tc.pass || (!tc.pass && err == nil) {
				t.Fatalf("pass=%v err=%v actual=%v", result.Pass, err, result.Actual)
			}
			if tc.pass && result.Actual == nil {
				t.Fatal("optional support classification was not retained")
			}
		})
	}
}

func TestChainInfoRetainsRemoteFiveSecondObservation(t *testing.T) {
	syncObject := map[string]any{"startingBlock": "0x0", "currentBlock": "0x1", "highestBlock": "0x2"}
	for _, tc := range []struct {
		name                    string
		id, sync, first, second any
		pass                    bool
	}{
		{"synced", "0x1", false, "0x1", "0x1", true},
		{"syncing object", "0x1", syncObject, "0x1", "0x2", true},
		{"zero chain id", "0x0", false, "0x1", "0x2", false},
		{"bad chain id", "bad", false, "0x1", "0x2", false},
		{"string false", "0x1", "false", "0x1", "0x2", false},
		{"true syncing", "0x1", true, "0x1", "0x2", false},
		{"empty syncing object", "0x1", map[string]any{}, "0x1", "0x2", false},
		{"bad syncing quantity", "0x1", map[string]any{"startingBlock": "0x0", "currentBlock": 1, "highestBlock": "0x2"}, "0x1", "0x2", false},
		{"genesis only", "0x1", false, "0x0", "0x0", false},
		{"height regression", "0x1", false, "0x2", "0x1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reads atomic.Int64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     any
					Method string
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				var value any
				switch req.Method {
				case "eth_chainId":
					value = tc.id
				case "eth_syncing":
					value = tc.sync
				case "eth_blockNumber":
					if reads.Add(1) == 1 {
						value = tc.first
					} else {
						value = tc.second
					}
				default:
					t.Error("unexpected method", req.Method)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": value})
			}))
			defer srv.Close()
			d := deps()
			a, ok := d.Actions.Assertion("chainInfo")
			if !ok {
				t.Fatal("missing public chainInfo observation")
			}
			started := time.Now()
			result, err := a.Check(t.Context(), &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}}, Spec: map[string]any{}})
			if result.Pass != tc.pass || (!tc.pass && err == nil) {
				t.Fatalf("pass=%v err=%v actual=%v", result.Pass, err, result.Actual)
			}
			if tc.pass && (time.Since(started) < 5*time.Second || reads.Load() != 2) {
				t.Fatal("original five-second pair was not observed", reads.Load(), time.Since(started))
			}
		})
	}
}

func TestChainInfoCancellationDoesNotClaimObservation(t *testing.T) {
	srv := mockRPC(t, map[string]any{"eth_chainId": "0x1", "eth_syncing": false, "eth_blockNumber": "0x1"})
	d := deps()
	a, ok := d.Actions.Assertion("chainInfo")
	if !ok {
		t.Fatal("missing public chainInfo observation")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	result, err := a.Check(ctx, &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}}, Spec: map[string]any{}})
	if result.Pass || err == nil {
		t.Fatal("cancelled observation accepted", result, err)
	}
}

func TestRemoteCasesKeepDistinctOriginalContracts(t *testing.T) {
	base := "../../tests/tc/go-stablenet/testnet/"
	for _, name := range []string{"06-remote-balance-query.json", "07-remote-chain-info.json", "08-remote-rpc-health.json", "09-remote-unlocked-transfer.json"} {
		raws, err := dsl.ReadFiles([]string{base + name})
		if err != nil {
			t.Fatal(err)
		}
		s, err := dsl.ParseV2(raws[0])
		if err != nil {
			t.Fatal(err)
		}
		if unused := IgnoredArguments(s); len(unused) > 0 {
			t.Fatal(name, unused)
		}
		if unresolved := interp.Unresolved(s, Registry()); len(unresolved) > 0 {
			t.Fatal(name, unresolved)
		}
		if name != "09-remote-unlocked-transfer.json" {
			continue
		}
		submitted, waited := false, false
		for _, st := range s.Sequence {
			if st.Do == "sendTx" {
				t.Fatal("remote node signing silently became local signing")
			}
			if st.Do == "read" && st.Args["method"] == "eth_sendTransaction" {
				params := st.Args["params"].([]any)
				tx := params[0].(map[string]any)
				submitted = tx["from"] == "$remoteSender" && tx["to"] == "$remoteRecipient" && tx["value"] == "0x1"
			}
			if st.Do == "waitFor" {
				waited = st.Args["source"] == "txStatus" && st.Args["hash"] == "$remoteHash" && st.Args["timeout"] == "60s"
			}
		}
		if !submitted || !waited {
			t.Fatal("remote original submission/receipt contract lost")
		}
	}
}

func TestRPCHealthClassifiesSupportAndVisitsEachNode(t *testing.T) {
	first := mockRPCAnswers(t, map[string]rpcAnswer{"eth_blockNumber": {result: "0x1"}})
	second := mockRPCAnswers(t, map[string]rpcAnswer{"eth_blockNumber": {result: "invalid"}})
	d := deps()
	a, _ := d.Actions.Assertion("rpcHealth")
	result, err := a.Check(t.Context(), &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: first.URL}, {Index: 2, RPCURL: second.URL}}, Spec: map[string]any{}})
	if result.Pass || err == nil {
		t.Fatal("unhealthy second endpoint was ignored", result, err)
	}
	actual := result.Actual.(map[string]any)
	if len(actual) != 2 {
		t.Fatal("selected endpoint results were omitted", actual)
	}
	one := actual["node1"].(map[string]any)
	for _, method := range []string{"net_peerCount", "web3_clientVersion"} {
		if one[method].(map[string]any)["status"] != "unsupported" {
			t.Fatal("absence classification missing", one)
		}
	}
}

func TestRemoteBalanceChecksAddressBeforeIO(t *testing.T) {
	for _, address := range []any{"0x", "0x1234", "0xgg00000000000000000000000000000000000000", "", nil, 17} {
		t.Run(fmt.Sprint(address), func(t *testing.T) {
			var calls atomic.Int64
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { calls.Add(1); _, _ = w.Write([]byte(`{"result":"0x0"}`)) }))
			defer srv.Close()
			d := deps()
			a, ok := d.Actions.Assertion("remoteBalance")
			if !ok {
				t.Fatal("missing strict remote balance query")
			}
			result, err := a.Check(t.Context(), &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}}, Spec: map[string]any{"address": address}})
			if result.Pass || err == nil || calls.Load() != 0 {
				t.Fatal("invalid remote address reached RPC or passed", result, err, calls.Load())
			}
		})
	}
}

func TestRemoteBalanceKeepsArbitraryAddressAndExactQuantity(t *testing.T) {
	address := "0x1234567890abcdef1234567890abcdef12345678"
	for _, tc := range []struct {
		value any
		pass  bool
	}{{"0x0", true}, {"0x123456789abcdef0123456789abcdef", true}, {"bad", false}, {nil, false}, {17, false}} {
		t.Run(fmt.Sprint(tc.value), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var req struct {
					ID     any
					Method string
					Params []any
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Error(err)
					return
				}
				if req.Method != "eth_getBalance" || len(req.Params) != 2 || req.Params[0] != address || req.Params[1] != "latest" {
					t.Error("arbitrary query address was replaced", req)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": tc.value})
			}))
			defer srv.Close()
			d := deps()
			a, _ := d.Actions.Assertion("remoteBalance")
			result, err := a.Check(t.Context(), &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}}, Spec: map[string]any{"address": address}})
			if result.Pass != tc.pass || (!tc.pass && err == nil) {
				t.Fatal("wrong balance classification", result, err)
			}
			if tc.pass {
				observed := result.Actual.(map[string]any)["node1"].(map[string]any)
				if observed["address"] != address || observed["balance"] != tc.value {
					t.Fatal("exact query result not retained", observed)
				}
			}
		})
	}
}
