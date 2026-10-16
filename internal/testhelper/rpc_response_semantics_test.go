package testhelper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestMethodPresentNeedsValidNodeResponse(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		pass, err  bool
	}{
		{"value", `{"result":"0x"}`, 200, true, false},
		{"invalid argument", `{"error":{"code":-32602,"message":"invalid argument"}}`, 200, true, false},
		{"method absent", `{"error":{"code":-32601,"message":"method not found"}}`, 200, false, false},
		{"RPC parse failure", `{"error":{"code":-32700,"message":"parse error"}}`, 200, false, true},
		{"RPC invalid request", `{"error":{"code":-32600,"message":"invalid request"}}`, 200, false, true},
		{"RPC internal failure", `{"error":{"code":-32603,"message":"internal error"}}`, 200, false, true},
		{"HTTP failure", `{"result":"0x"}`, 503, false, true},
		{"invalid JSON", `not JSON`, 200, false, true},
		{"empty response", `{}`, 200, false, true},
		{"contradictory response", `{"result":"0xaccepted","error":{"code":-32000,"message":"insufficient funds"}}`, 200, false, true},
		{"incomplete error", `{"error":{"message":"insufficient funds"}}`, 200, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			d := deps()
			as, _ := d.Actions.Assertion(assertMethodPresent)
			r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}}, Spec: map[string]any{"method": "eth_signRawFeeDelegateTransaction"}})
			if r.Pass != tc.pass || (err != nil) != tc.err {
				t.Fatalf("pass=%v err=%v; want pass=%v error=%v", r.Pass, err, tc.pass, tc.err)
			}
		})
	}
}

func TestRPCErrorRejectsContradictoryEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":"0xaccepted","error":{"code":-32000,"message":"insufficient funds"}}`))
	}))
	defer srv.Close()
	r, err := checkRPCError(t, srv.URL, map[string]any{"method": "eth_sendRawTransaction", "reason": "insufficient funds"})
	if r.Pass || err == nil {
		t.Fatalf("contradictory response became rejection evidence: pass=%v err=%v", r.Pass, err)
	}
}
