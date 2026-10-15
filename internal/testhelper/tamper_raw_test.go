package testhelper

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func rawTamperFixture(t *testing.T) string {
	t.Helper()
	s, _, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	p, _, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	one := big.NewInt(1)
	raw, err := accounts.EncodeFeeDelegatedTampered(s, p, testTo, one, 8283, 0, one, one, "sender")
	if err != nil {
		t.Fatal(err)
	}
	return "0x" + hex.EncodeToString(raw)
}

func TestTamperRawInputUsesOnlyRawSubmission(t *testing.T) {
	for _, which := range []string{"sender", "feepayer"} {
		t.Run(which, func(t *testing.T) {
			raw := rawTamperFixture(t)
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				var q struct {
					ID     any
					Method string
					Params []string
				}
				if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
					t.Error(err)
					return
				}
				if q.Method != "eth_sendRawTransaction" || len(q.Params) != 1 || q.Params[0] == raw || len(q.Params[0]) != len(raw) || !strings.HasPrefix(q.Params[0], "0x16") {
					t.Error("raw mode rebuilt or failed to mutate the supplied transaction", q.Method)
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"id": q.ID, "error": map[string]any{"code": -32000, "message": "fee delegation: invalid feePayer"}})
			}))
			defer server.Close()
			d := deps()
			d.Accounts = tamperProvider{}
			ac := &interp.ActionCtx{Deps: &d, Env: envWithNode(t, server.URL), Args: map[string]any{"raw": raw, "which": which, "reason": "invalid feePayer"}}
			if err := (sendRawTamperedAction{}).Do(context.Background(), ac); err != nil {
				t.Fatal("valid raw input must reach submission without builder keys or preparation queries", err)
			}
			if calls.Load() != 1 || ac.Hash != "" || ac.Value == nil {
				t.Fatal("missing exact raw rejection", calls.Load(), ac.Value)
			}
		})
	}
}

func TestTamperRawRejectsAmbiguousOrMalformedInputBeforeRPC(t *testing.T) {
	for _, field := range []string{"senderKey", "feePayerKey", "to", "value", "empty", "nil", "wrong type", "bad hex", "wrong envelope"} {
		t.Run(field, func(t *testing.T) {
			args := map[string]any{"which": "sender", "raw": rawTamperFixture(t)}
			switch field {
			case "empty":
				args["raw"] = ""
			case "nil":
				args["raw"] = nil
			case "wrong type":
				args["raw"] = true
			case "bad hex":
				args["raw"] = "0x16zz"
			case "wrong envelope":
				args["raw"] = "0x02c0"
			default:
				args[field] = "1"
			}
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls.Add(1) }))
			defer server.Close()
			d := deps()
			d.Accounts = tamperProvider{}
			err := (sendRawTamperedAction{}).Do(context.Background(), &interp.ActionCtx{Deps: &d, Env: envWithNode(t, server.URL), Args: args})
			if err == nil || !strings.Contains(err.Error(), "raw") || calls.Load() != 0 {
				t.Fatalf("invalid raw input reached RPC or lacked a raw-specific error: %v, calls=%d", err, calls.Load())
			}
		})
	}
}
