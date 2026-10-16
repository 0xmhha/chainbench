package testhelper

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
	"golang.org/x/crypto/sha3"
)

func TestPublicAuthorizationCostsRequireDeployedDelegate(t *testing.T) {
	fixture, err := os.ReadFile("testdata/simple7702-runtime.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, fault := range []string{"valid", "empty code", "STOP code", "wrong code hash", "wrong getter", "code RPC error", "baseline too low", "baseline too high", "one auth too low", "one auth too high", "two auth too low", "two auth too high"} {
		t.Run(fault, func(t *testing.T) {
			sponsor, delegate := "0x"+strings.Repeat("12", 20), "0x"+strings.Repeat("34", 20)
			var mu sync.Mutex
			var protocol []string
			estimates, codes, hashes, getters := 0, 0, 0, 0
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
					codes++
					if len(q.Params) != 2 || q.Params[0] != delegate || q.Params[1] != "latest" {
						protocol = append(protocol, "code must belong to the deployed delegate")
					}
					result = strings.TrimSpace(string(fixture))
					switch fault {
					case "empty code":
						result = "0x"
					case "STOP code":
						result = "0x00"
					case "code RPC error":
						response["error"] = map[string]any{"code": -32000, "message": "code unavailable"}
					}
				case "web3_sha3":
					hashes++
					code, _ := q.Params[0].(string)
					b, err := hex.DecodeString(strings.TrimPrefix(code, "0x"))
					if err != nil {
						t.Error(err)
						return
					}
					h := sha3.NewLegacyKeccak256()
					_, _ = h.Write(b)
					result = fmt.Sprintf("0x%x", h.Sum(nil))
					if fault == "wrong code hash" {
						result = "0x" + strings.Repeat("0", 64)
					}
				case "eth_call":
					getters++
					call, _ := q.Params[0].(map[string]any)
					h := sha3.NewLegacyKeccak256()
					_, _ = h.Write([]byte("getNumber()"))
					selector := fmt.Sprintf("0x%x", h.Sum(nil)[:4])
					if call["to"] != delegate || call["data"] != selector {
						protocol = append(protocol, "wrong delegate getter")
					}
					result = "0x" + strings.Repeat("0", 64)
					if fault == "wrong getter" {
						result = "0x" + strings.Repeat("0", 63) + "1"
					}
				case "eth_estimateGas":
					estimates++
					tx := q.Params[0].(map[string]any)
					if tx["from"] != sponsor || tx["to"] != sponsor || tx["value"] != nil {
						protocol = append(protocol, "original17 requires same sponsor and omitted value")
					}
					auths, _ := tx["authorizationList"].([]any)
					n := len(auths)
					gas := 21000 + n*25000
					if (fault == "baseline too low" && n == 0) || (fault == "one auth too low" && n == 1) || (fault == "two auth too low" && n == 2) {
						gas--
					}
					if (fault == "baseline too high" && n == 0) || (fault == "one auth too high" && n == 1) || (fault == "two auth too high" && n == 2) {
						gas = []int{21001, 47501, 74001}[n]
					}
					result = fmt.Sprintf("0x%x", gas)
				default:
					t.Error("unexpected observation", q.Method)
				}
				if response["error"] == nil {
					response["result"] = result
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			raws, err := dsl.ReadFiles([]string{"../../tests/tc/go-stablenet/post-v1.0.0-change/common-all/17b-estimategas-deployed-authorizationlist.json"})
			if err != nil {
				t.Fatal(err)
			}
			s, err := dsl.ParseV2(raws[0])
			if err != nil {
				t.Fatal(err)
			}
			// Deployment, account creation and signing are fixture setup. Native
			// proof runs them too; retain every public observation and assertion.
			selected := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Do == "newAccount" || st.Do == "deployContract" || st.Do == "signAuthorization" || (st.Do == "read" && st.Args["method"] == "eth_coinbase") {
					continue
				}
				selected = append(selected, st)
			}
			s.Sequence = selected
			s.EnvValues = map[string]any{"sponsor": sponsor, "delegate": delegate, "auth1": map[string]any{"nonce": "0x0"}, "auth2": map[string]any{"nonce": "0x1"}}
			got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), &recordStub{})
			want := session.StatusFail
			if fault == "valid" {
				want = session.StatusPass
			}
			mu.Lock()
			defer mu.Unlock()
			if err != nil || got != want {
				t.Fatalf("fault %q status=%s want=%s err=%v", fault, got, want, err)
			}
			if fault == "valid" && (estimates != 3 || codes != 1 || hashes != 1 || getters != 1 || len(protocol) != 0) {
				t.Fatalf("missing deployed-code evidence: estimates=%d codes=%d hashes=%d getters=%d issues=%v", estimates, codes, hashes, getters, protocol)
			}
		})
	}
}
