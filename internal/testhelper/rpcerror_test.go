package testhelper

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// rpcAnswer is what a mock node answers one method with: a result, or an error.
type rpcAnswer struct {
	result  any
	code    int
	message string
}

func mockRPCAnswers(t *testing.T, answers map[string]rpcAnswer) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		a, ok := answers[req.Method]
		switch {
		case !ok:
			resp["error"] = map[string]any{"code": -32601, "message": "the method " + req.Method + " does not exist/is not available"}
		case a.message != "":
			resp["error"] = map[string]any{"code": a.code, "message": a.message}
		default:
			resp["result"] = a.result
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestRPCError: rpcError passes only when the node answers the call with an
// error that says what the case expects. A value, another reason, or a method
// the node lacks each fail it; a node that cannot be reached is an error, not a
// verdict, since it says nothing about what the node would have answered.
func TestRPCError(t *testing.T) {
	const method = "eth_signRawFeeDelegateTransaction"
	cases := []struct {
		name     string
		answer   *rpcAnswer
		reason   string
		wantPass bool
		wantErr  bool
	}{
		{"refused for the named reason", &rpcAnswer{code: -32000, message: "missing FeePayer"}, "missing feepayer", true, false},
		{"refused for another reason", &rpcAnswer{code: -32000, message: "senderTx type error"}, "missing FeePayer", false, false},
		{"answered with a value", &rpcAnswer{result: map[string]any{"raw": "0x01"}}, "missing FeePayer", false, false},
		{"method absent", nil, "missing FeePayer", false, false},
		{"no reason named", &rpcAnswer{code: -32000, message: "missing FeePayer"}, "", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			answers := map[string]rpcAnswer{}
			if tc.answer != nil {
				answers[method] = *tc.answer
			}
			srv := mockRPCAnswers(t, answers)
			r, err := checkRPCError(t, srv.URL, map[string]any{"method": method, "params": []any{map[string]any{}, "0x02"}, "reason": tc.reason})
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tc.wantErr)
			}
			if r.Pass != tc.wantPass {
				t.Errorf("pass = %v, want %v (actual %v)", r.Pass, tc.wantPass, r.Actual)
			}
		})
	}
}

func TestRPCError_UnreachableNodeIsAnError(t *testing.T) {
	srv := mockRPCAnswers(t, nil)
	url := srv.URL
	srv.Close()
	r, err := checkRPCError(t, url, map[string]any{"method": "eth_x", "reason": "anything"})
	if err == nil || r.Pass {
		t.Fatalf("an unreachable node gave pass=%v err=%v, want an error", r.Pass, err)
	}
}

func checkRPCError(t *testing.T, url string, spec map[string]any) (session.AssertResult, error) {
	t.Helper()
	d := deps()
	as, ok := d.Actions.Assertion(assertRPCError)
	if !ok {
		t.Fatalf("%s is not registered", assertRPCError)
	}
	spec["assert"] = assertRPCError
	return as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: url}}, Spec: spec})
}
