package testhelper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestPublicAuthorizationEstimatesRequireEOATransferBaseline(t *testing.T) {
	for _, fault := range []string{"valid", "receiver STOP code", "receiver delegation code", "missing receiver code", "receiver code RPC error", "baseline too low", "baseline too high", "one auth too low", "one auth too high", "two auth too low", "two auth too high"} {
		t.Run(fault, func(t *testing.T) {
			sponsor, receiver := "0x"+strings.Repeat("12", 20), "0x"+strings.Repeat("34", 20)
			var mu sync.Mutex
			var protocol []string
			estimates, codeReads := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				var q struct {
					ID     any
					Method string
					Params []any
				}
				if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
					t.Error(err)
					return
				}
				var result any
				response := map[string]any{"jsonrpc": "2.0", "id": q.ID}
				switch q.Method {
				case "eth_getCode":
					codeReads++
					if len(q.Params) != 2 || q.Params[0] != receiver || q.Params[1] != "latest" {
						protocol = append(protocol, "code must belong to receiver at latest")
					}
					result = "0x"
					switch fault {
					case "receiver STOP code":
						result = "0x00"
					case "receiver delegation code":
						result = "0xef0100" + strings.Repeat("56", 20)
					case "missing receiver code":
						result = nil
					case "receiver code RPC error":
						response["error"] = map[string]any{"code": -32000, "message": "code unavailable"}
					}
				case "eth_estimateGas":
					estimates++
					if len(q.Params) != 1 {
						t.Error("expected one transaction object")
						return
					}
					tx, ok := q.Params[0].(map[string]any)
					if !ok {
						t.Error("expected transaction object")
						return
					}
					if tx["from"] != sponsor || tx["to"] != receiver || tx["value"] != "0x1" || tx["from"] == tx["to"] {
						protocol = append(protocol, "all estimates must preserve distinct sender/receiver and one wei")
					}
					auths, _ := tx["authorizationList"].([]any)
					n := len(auths)
					gas := 21000 + n*25000
					if (fault == "baseline too low" && n == 0) || (fault == "one auth too low" && n == 1) || (fault == "two auth too low" && n == 2) {
						gas--
					}
					if (fault == "baseline too high" && n == 0) || (fault == "one auth too high" && n == 1) || (fault == "two auth too high" && n == 2) {
						gas = []int{21001, 46691, 72066}[n]
					}
					result = fmt.Sprintf("0x%x", gas)
				default:
					t.Error("unexpected public observation", q.Method)
				}
				if response["error"] == nil {
					response["result"] = result
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			raws, err := dsl.ReadFiles([]string{"../../tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json"})
			if err != nil {
				t.Fatal(err)
			}
			s, err := dsl.ParseV2(raws[0])
			if err != nil {
				t.Fatal(err)
			}
			// Account creation and signing are fixture setup, covered by native
			// execution. Keep every public observation, assertion and bound.
			selected := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Do == "newAccount" || st.Do == "signAuthorization" || (st.Do == "read" && st.Args["method"] == "eth_coinbase") {
					continue
				}
				selected = append(selected, st)
			}
			s.Sequence = selected
			s.EnvValues = map[string]any{"sponsor": sponsor, "receiver": receiver, "auth1": map[string]any{"nonce": "0x0"}, "auth2": map[string]any{"nonce": "0x1"}}
			rec := &recordStub{}
			got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			want := session.StatusFail
			if fault == "valid" {
				want = session.StatusPass
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil || got != want {
				t.Fatalf("fault %q status=%s, want %s: err=%v assertions=%+v", fault, got, want, err, rec.asserts)
			}
			if fault == "valid" && (estimates != 3 || codeReads != 1 || len(protocol) != 0) {
				t.Fatalf("baseline protocol changed: estimates=%d codeReads=%d issues=%v", estimates, codeReads, protocol)
			}
		})
	}
}
