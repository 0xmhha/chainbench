package testhelper

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestSubmitRejectionRequiresNodeSubmissionError(t *testing.T) {
	raw := &rpc.ServerError{Method: "eth_sendRawTransaction", Code: -32000, Message: "nonce too low"}
	cases := []struct {
		name, hash string
		err        error
		reason     string
		pass       bool
	}{
		{"raw submission", "", raw, "NONCE TOO LOW", true},
		{"wrapped submission", "", fmt.Errorf("send: %w", raw), "nonce", true},
		{"node submission", "", &rpc.ServerError{Method: "eth_sendTransaction", Code: -32000, Message: "insufficient funds"}, "funds", true},
		{"accepted", "0xaccepted", nil, "", false},
		{"hash and error", "0xaccepted", raw, "", false},
		{"estimation", "", &rpc.ServerError{Method: "eth_estimateGas", Code: -32000, Message: "nonce too low"}, "nonce", false},
		{"nonce lookup", "", &rpc.ServerError{Method: "eth_getTransactionCount", Code: -32000, Message: "nonce too low"}, "", false},
		{"local guard", "", errors.New("sender is blacklisted"), "blacklisted", false},
		{"transport", "", errors.New("connection refused: nonce too low"), "nonce", false},
		{"untyped message", "", errors.New("eth_sendRawTransaction: rpc error -32000: nonce too low"), "nonce", false},
		{"method is not a reason", "", raw, "eth_sendRawTransaction", false},
		{"different reason", "", raw, "insufficient funds", false},
	}
	for _, code := range []int{-32700, -32600, -32601, -32602, -32603} {
		cases = append(cases, struct {
			name, hash string
			err        error
			reason     string
			pass       bool
		}{fmt.Sprintf("protocol error %d", code), "", &rpc.ServerError{Method: "eth_sendRawTransaction", Code: code, Message: "invalid transaction request"}, "", false})
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ac := &interp.ActionCtx{Args: map[string]any{"reason": c.reason}}
			err := checkSubmitRejected(c.hash, c.err, ac)
			if (err == nil) != c.pass {
				t.Fatalf("pass=%v, want=%v: %v", err == nil, c.pass, err)
			}
			if c.pass && ac.Value != c.err.Error() {
				t.Fatalf("rejection evidence changed: %v", ac.Value)
			}
			if !c.pass && ac.Value != nil {
				t.Fatalf("non-submission error was bound as rejection evidence: %v", ac.Value)
			}
		})
	}
}

func TestKeptOutDoesNotPassPreparationOrTransportErrors(t *testing.T) {
	for _, err := range []error{errors.New("connection refused"), &rpc.ServerError{Method: "eth_estimateGas", Code: -32000, Message: "fee cap too low"}} {
		ac := &interp.ActionCtx{Args: map[string]any{}}
		if got := checkKeptOut(context.Background(), nil, "", err, ac); got == nil {
			t.Fatalf("keptOut passed without a submission: %v", err)
		}
	}
}
