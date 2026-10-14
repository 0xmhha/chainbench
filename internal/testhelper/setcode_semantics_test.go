package testhelper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestSetCodeCaseRequiresDelegatedExecutionContext(t *testing.T) {
	const authority = "0x0000000000000000000000000000000000001234"
	const delegate = "0x1111111111111111111111111111111111111111"
	const runtime = "0x36602011600e57600035600055005b60005460005260206000f3"
	const marker = "0xef01001111111111111111111111111111111111111111"
	zero := "0x" + strings.Repeat("0", 64)
	fortyTwo := "0x" + strings.Repeat("0", 62) + "2a"
	for _, tc := range []struct {
		name, returned, authoritySlot, delegateSlot string
		want                                        session.TestStatus
	}{
		{"authority storage executes", fortyTwo, fortyTwo, zero, session.StatusPass},
		{"marker without execution", zero, zero, zero, session.StatusFail},
		{"wrong delegate storage context", zero, zero, fortyTwo, session.StatusFail},
		{"both storage contexts changed", fortyTwo, fortyTwo, fortyTwo, session.StatusFail},
		{"return42 without committed storage", fortyTwo, zero, zero, session.StatusFail},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				switch q.Method {
				case "eth_getCode":
					if q.Params[0] == authority {
						result = marker
					} else {
						result = runtime
					}
				case "eth_getTransactionReceipt":
					result = map[string]any{"status": "0x1"}
				case "eth_getTransactionByHash":
					result = map[string]any{"type": "0x4"}
				case "eth_call":
					args := q.Params[0].(map[string]any)
					if args["to"] == authority {
						result = tc.returned
					} else {
						result = tc.delegateSlot
					}
				case "eth_getStorageAt":
					if q.Params[0] == authority {
						result = tc.authoritySlot
					} else {
						result = tc.delegateSlot
					}
				default:
					t.Errorf("unexpected method %s", q.Method)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
			}))
			defer srv.Close()
			raw, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json"}, "../../presets/chain/wbft-bp4-en1.json")
			if err != nil {
				t.Fatal(err)
			}
			s, err := dsl.ParseV2(raw[0])
			if err != nil {
				t.Fatal(err)
			}
			selected := []dsl.Statement{}
			active := false
			for _, st := range s.Sequence {
				if st.Do == "sendSetCode" {
					active = true
					continue
				}
				if st.Do == "sendTx" && st.Args["save"] == "writeHash" {
					selected = nil
					active = true
					continue
				}
				if !active || st.Do != "" {
					continue
				}
				delete(st.Args, "on")
				selected = append(selected, st)
			}
			s.Sequence = selected
			s.EnvValues = map[string]any{"authority": authority, "delegate": delegate, "marker": marker, "fundHash": "fund", "deployHash": "deploy", "scHash": "setcode", "writeHash": "write"}
			rec := &recordStub{}
			got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			if err != nil || got != tc.want {
				t.Fatalf("public set-code case status=%s err=%v want=%s; assertions=%+v", got, err, tc.want, rec.asserts)
			}
		})
	}
}
