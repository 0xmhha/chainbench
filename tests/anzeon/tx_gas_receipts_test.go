package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

var txGasReceiptCases = []string{
	"gaslimit-exceeded-rejected",
	"revert-tx-status-zero",
	"out-of-gas-consumes-all",
}

func TestTxGasReceiptCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range txGasReceiptCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

func TestTxGasReceiptCases_SkipForeignChain(t *testing.T) {
	ns, _ := attach.Build("wemix", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: txGasReceiptCases})
	if len(rep.Results) != len(txGasReceiptCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(txGasReceiptCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on wemix", r.Name, r.Status)
		}
	}
}
