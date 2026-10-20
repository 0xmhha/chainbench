package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestPublicPoolContentBindsSubmittedHashes(t *testing.T) {
	for _, name := range []string{"valid", "wrong pending hash", "wrong queued hash", "swapped buckets"} {
		t.Run(name, func(t *testing.T) {
			pending := map[string]any{}
			queued := map[string]any{}
			for _, n := range []int{0, 1, 3, 4} {
				tx := map[string]any{"hash": fmt.Sprint("h", n), "nonce": fmt.Sprintf("0x%x", n), "from": "0x0000000000000000000000000000000000001234", "gas": "0x5208", "gasPrice": "0x64"}
				if n < 2 {
					pending[fmt.Sprint(n)] = tx
				} else {
					queued[fmt.Sprint(n)] = tx
				}
			}
			if name == "wrong pending hash" {
				pending["0"].(map[string]any)["hash"] = "unrelated"
			}
			if name == "wrong queued hash" {
				queued["4"].(map[string]any)["hash"] = "unrelated"
			}
			if name == "swapped buckets" {
				pending, queued = queued, pending
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct{ ID any }
				_ = json.NewDecoder(r.Body).Decode(&q)
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": map[string]any{"pending": pending, "queued": queued}})
			}))
			defer srv.Close()
			raws, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/rpc/CT-RPC-012-txpool-content-well-formed.json"}, "../../presets/chain/stablenet-bp4.json")
			if err != nil {
				t.Fatal(err)
			}
			s, err := dsl.ParseV2(raws[0])
			if err != nil {
				t.Fatal(err)
			}
			selected := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Expect == "rpcCall" && st.Args["method"] == "txpool_contentFrom" {
					delete(st.Args, "on")
					selected = append(selected, st)
				}
			}
			s.Sequence = selected
			s.EnvValues = map[string]any{"spender": "0x0000000000000000000000000000000000001234", "h0": "h0", "h1": "h1", "h3": "h3", "h4": "h4"}
			rec := &recordStub{}
			got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			want := session.StatusFail
			if name == "valid" {
				want = session.StatusPass
			}
			if err != nil || got != want {
				t.Fatalf("public pool %s err=%v want=%s assertions=%+v", got, err, want, rec.asserts)
			}
		})
	}
}

func TestIsolatedENPoolCaseConsumesAllInputs(t *testing.T) {
	raws, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/rpc/CT-RPC-016-isolated-en-pending-queued.json"}, "../../presets/chain/wbft-bp4-en1.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := dsl.ParseV2(raws[0])
	if err != nil {
		t.Fatal(err)
	}
	if ignored := IgnoredArguments(s); len(ignored) > 0 {
		t.Fatalf("ignored inputs: %v", ignored)
	}
}

func TestIsolatedENPoolUsesCurrentFeeQuotes(t *testing.T) {
	srv := mockRPC(t, map[string]any{"eth_gasPrice": "0x2b4abd176000", "eth_maxPriorityFeePerGas": "0x191a20322000"})
	raws, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/rpc/CT-RPC-016-isolated-en-pending-queued.json"}, "../../presets/chain/stablenet-bp4-en1.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := dsl.ParseV2(raws[0])
	if err != nil {
		t.Fatal(err)
	}
	selected := []dsl.Statement{}
	for _, st := range s.Sequence {
		if st.Do == "partition" {
			break
		}
		if st.Do == "read" {
			delete(st.Args, "on")
			selected = append(selected, st)
		}
	}
	// A locally admitted 35e12 cap cannot pay this node's observed 47.6e12
	// base+tip requirement when another peer validates it as a remote transaction.
	selected = append(selected, dsl.Statement{Expect: "derive", Args: map[string]any{"op": "sum", "of": []any{"$poolFeeCap"}, "compare": "GreaterOrEqual", "expected": "47600000000000"}})
	s.Sequence = selected
	rec := &recordStub{}
	got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
	if err != nil || got != session.StatusPass {
		t.Fatalf("current remote fee quote not covered: %s err=%v assertions=%+v", got, err, rec.asserts)
	}
}
