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

// nonceCase loads the public case so mutation checks exercise its actual assertions.
func nonceCase(t *testing.T, name string) dsl.Spec {
	t.Helper()
	raws, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/tx/" + name}, "../../presets/chain/stablenet-bp4.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := dsl.ParseV2(raws[0])
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNonceCaseRejectsReversedMinedOrder(t *testing.T) {
	for _, tc := range []struct {
		name            string
		blocks, indices [3]int
		want            session.TestStatus
	}{
		{"same block", [3]int{10, 10, 10}, [3]int{0, 1, 2}, session.StatusPass},
		{"different blocks", [3]int{10, 11, 12}, [3]int{5, 0, 0}, session.StatusPass},
		{"reversed indices", [3]int{10, 10, 10}, [3]int{2, 1, 0}, session.StatusFail},
		{"duplicate position", [3]int{10, 10, 10}, [3]int{0, 0, 1}, session.StatusFail},
		{"reversed blocks", [3]int{12, 11, 10}, [3]int{0, 1, 2}, session.StatusFail},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct {
					ID     any
					Method string
					Params []string
				}
				if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
					t.Error(err)
					return
				}
				var result any = "0x3"
				if q.Method == "eth_getTransactionReceipt" || q.Method == "eth_getTransactionByHash" {
					n := 0
					if len(q.Params) > 0 {
						for j := 0; j < 3; j++ {
							if q.Params[0] == fmt.Sprintf("h%d", j) {
								n = j
							}
						}
					}
					result = map[string]any{"status": "0x1", "nonce": fmt.Sprintf("0x%x", n), "blockNumber": fmt.Sprintf("0x%x", tc.blocks[n]), "transactionIndex": fmt.Sprintf("0x%x", tc.indices[n])}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
			}))
			defer srv.Close()
			s := nonceCase(t, "CT-TX-012-nonce-ordering.json")
			start := 0
			for j, st := range s.Sequence {
				if st.Do == "waitFor" {
					start = j + 1
				}
			}
			s.Sequence = s.Sequence[start:]
			s.EnvValues = map[string]any{"acct": "0x0000000000000000000000000000000000001234", "fundHash": "fund", "h0": "h0", "h1": "h1", "h2": "h2"}
			for _, st := range s.Sequence {
				delete(st.Args, "on")
			}
			rec := &recordStub{}
			got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			if err != nil || got != tc.want {
				t.Fatalf("public case status=%s err=%v, want %s; assertions=%+v", got, err, tc.want, rec.asserts)
			}
		})
	}
}

func TestReplacementCaseObservesQueuedHash(t *testing.T) {
	for _, phase := range []string{"tx1", "tx2"} {
		for _, hash := range []string{phase, "wrong"} {
			t.Run(phase+"/"+hash, func(t *testing.T) {
				s := nonceCase(t, "CT-TX-013-same-nonce-replacement.json")
				selected := []dsl.Statement{}
				active := false
				for _, st := range s.Sequence {
					if st.Do == "sendTx" {
						active = st.Args["save"] == phase
						continue
					}
					if active && st.Expect == "rpcCall" && st.Args["method"] == "txpool_contentFrom" {
						delete(st.Args, "on")
						selected = append(selected, st)
					}
				}
				if len(selected) == 0 {
					t.Fatal("public case never observes the queued replacement phase")
				}
				s.Sequence = selected
				s.EnvValues = map[string]any{"acct": "0x0000000000000000000000000000000000001234", "tx1": "tx1", "tx2": "tx2"}
				srv := mockRPC(t, map[string]any{"txpool_contentFrom": map[string]any{"pending": map[string]any{}, "queued": map[string]any{"2": map[string]any{"hash": hash}}}})
				got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), &recordStub{})
				want := session.StatusFail
				if hash == phase {
					want = session.StatusPass
				}
				if err != nil || got != want {
					t.Fatalf("queued snapshot status=%s err=%v, want %s", got, err, want)
				}
			})
		}
	}
}
