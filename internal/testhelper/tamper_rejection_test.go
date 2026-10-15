package testhelper

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type tamperProvider struct{ accounts.AccountProvider }

func (tamperProvider) SupportsTxType(kind byte) bool { return kind == 0x16 }

func TestTamperRequiresActualNodeSubmissionRejection(t *testing.T) {
	sender, _, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	payer, _, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	const reason = "fee delegation: invalid feePayer"
	cases := []struct {
		name, mode, failedMethod, reason string
		code                             int
		pass                             bool
	}{
		{"actual rejection", "node", "eth_sendRawTransaction", reason, -32000, true},
		{"case insensitive reason", "node", "eth_sendRawTransaction", strings.ToUpper(reason), -32000, true},
		{"accepted", "accepted", "eth_sendRawTransaction", "", 0, false},
		{"http", "http", "eth_sendRawTransaction", "", 0, false},
		{"malformed json", "malformed", "eth_sendRawTransaction", "", 0, false},
		{"wrong result type", "shape", "eth_sendRawTransaction", "", 0, false},
		{"closed connection", "disconnect", "eth_sendRawTransaction", "", 0, false},
		{"method name is not reason", "node", "eth_sendRawTransaction", "eth_sendRawTransaction", -32000, false},
		{"different rejection", "node", "eth_sendRawTransaction", "insufficient funds", -32000, false},
		{"chain lookup", "node", "eth_chainId", reason, -32000, false},
		{"nonce lookup", "node", "eth_getTransactionCount", reason, -32000, false},
		{"fee lookup", "node", "eth_getBlockByNumber", reason, -32000, false},
	}
	for _, code := range []int{-32700, -32600, -32601, -32602, -32603} {
		cases = append(cases, struct {
			name, mode, failedMethod, reason string
			code                             int
			pass                             bool
		}{fmt.Sprintf("protocol %d", code), "node", "eth_sendRawTransaction", reason, code, false})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var submissions atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct {
					ID     any
					Method string
					Params []string
				}
				// Only submission has string-only params; decode the envelope separately.
				var envelope struct {
					ID     any
					Method string
					Params json.RawMessage
				}
				if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
					t.Error(err)
					return
				}
				q.ID, q.Method = envelope.ID, envelope.Method
				if q.Method == "eth_sendRawTransaction" {
					submissions.Add(1)
					if err := json.Unmarshal(envelope.Params, &q.Params); err != nil || len(q.Params) != 1 || !strings.HasPrefix(q.Params[0], "0x16") {
						t.Error("submission must contain the actual encoded fee-delegated transaction", err)
						return
					}
				}
				if q.Method == c.failedMethod {
					switch c.mode {
					case "http":
						http.Error(w, reason, http.StatusServiceUnavailable)
					case "malformed":
						_, _ = w.Write([]byte(reason))
					case "disconnect":
						conn, _, e := w.(http.Hijacker).Hijack()
						if e != nil {
							t.Error(e)
							return
						}
						_ = conn.Close()
					case "shape":
						_ = json.NewEncoder(w).Encode(map[string]any{"id": q.ID, "result": map[string]any{"error": reason}})
					case "accepted":
						_ = json.NewEncoder(w).Encode(map[string]any{"id": q.ID, "result": "0xaccepted"})
					default:
						_ = json.NewEncoder(w).Encode(map[string]any{"id": q.ID, "error": map[string]any{"code": c.code, "message": reason}})
					}
					return
				}
				results := map[string]any{"eth_chainId": "0x205b", "eth_getTransactionCount": "0x0", "eth_getBlockByNumber": map[string]any{"number": "0x1", "baseFeePerGas": "0x1"}, "eth_maxPriorityFeePerGas": "0x1"}
				result, ok := results[q.Method]
				if !ok {
					t.Error("unexpected method", q.Method)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"id": q.ID, "result": result})
			}))
			defer server.Close()
			d := deps()
			d.Accounts = tamperProvider{}
			ac := &interp.ActionCtx{Deps: &d, Env: envWithNode(t, server.URL), Args: map[string]any{"senderKey": hex.EncodeToString(sender), "feePayerKey": hex.EncodeToString(payer), "to": testTo, "which": "sender", "reason": c.reason}}
			err := (sendRawTamperedAction{}).Do(context.Background(), ac)
			if (err == nil) != c.pass {
				t.Errorf("passed=%v, want=%v: %v", err == nil, c.pass, err)
			}
			if !c.pass && ac.Value != nil {
				t.Errorf("invalid outcome was saved as rejection evidence: %v", ac.Value)
			}
			wantSubmissions := 0
			if c.failedMethod == "eth_sendRawTransaction" {
				wantSubmissions = 1
			}
			if int(submissions.Load()) != wantSubmissions {
				t.Errorf("submissions=%d want %d", submissions.Load(), wantSubmissions)
			}
		})
	}
}
