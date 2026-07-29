package accounts_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the cases

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

var txRejectionCases = []string{
	"dynamic-fee-below-basefee-rejected",
	"gas-limit-exceeds-block-rejected",
	"out-of-order-nonces-mine",
	"same-nonce-replacement",
	"fee-delegated-unfunded-feepayer-rejected",
}

func TestTxRejectionCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range txRejectionCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

// These cases drive real rejections/mining, so they are only meaningful against
// a live wbft node. Off a foreign chain they must gate out cleanly.
func TestTxRejectionCases_SkipForeignChain(t *testing.T) {
	ns, _ := attach.Build("ethereum", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: txRejectionCases})
	if len(rep.Results) != len(txRejectionCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(txRejectionCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on ethereum", r.Name, r.Status)
		}
	}
}
