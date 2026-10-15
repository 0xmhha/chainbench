package testhelper

import (
	"fmt"
	"testing"
)

func TestRPCErrorSubmissionRequiresTransactionRefusal(t *testing.T) {
	for _, method := range []string{"eth_sendTransaction", "eth_sendRawTransaction"} {
		for _, code := range []int{-32700, -32600, -32601, -32602, -32603, -32000, 3} {
			t.Run(fmt.Sprintf("%s/%d", method, code), func(t *testing.T) {
				const reason = "insufficient funds"
				srv := mockRPCAnswers(t, map[string]rpcAnswer{method: {code: code, message: reason}})
				r, err := checkRPCError(t, srv.URL, map[string]any{"method": method, "params": []any{"0x02"}, "reason": reason})
				want := code == -32000 || code == 3
				if err != nil || r.Pass != want {
					t.Fatalf("RPC code %d counted as transaction refusal=%v, want %v: err=%v result=%+v", code, r.Pass, want, err, r)
				}
			})
		}
	}
}

func TestRPCErrorKeepsGenericArgumentValidation(t *testing.T) {
	const method = "eth_signRawFeeDelegateTransaction"
	const reason = "invalid feePayer argument"
	srv := mockRPCAnswers(t, map[string]rpcAnswer{method: {code: -32602, message: reason}})
	r, err := checkRPCError(t, srv.URL, map[string]any{"method": method, "params": []any{}, "reason": reason})
	if err != nil || !r.Pass {
		t.Fatalf("generic argument-error contract changed: err=%v result=%+v", err, r)
	}
}

func TestRPCErrorSubmissionKeepsMissingMethodGuard(t *testing.T) {
	for _, method := range []string{"eth_sendTransaction", "eth_sendRawTransaction"} {
		t.Run(method, func(t *testing.T) {
			const reason = "method not found"
			srv := mockRPCAnswers(t, map[string]rpcAnswer{method: {code: -32000, message: reason}})
			r, err := checkRPCError(t, srv.URL, map[string]any{"method": method, "reason": reason})
			if err != nil || r.Pass {
				t.Fatalf("missing method became a transaction refusal: err=%v result=%+v", err, r)
			}
		})
	}
}
