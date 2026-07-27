package accounts_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the cases

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// txErrorCases are the live error-path cases in this file. They drive real
// transactions/calls against a live node, so the sibling tests validate
// registration and chain gating only.
var txErrorCases = []string{
	"insufficient-funds-rejected",
	"eth-call-revert-returns-error",
}

func TestTxErrorCases_Register(t *testing.T) {
	registered := map[string]bool{}
	for _, c := range testkit.Cases() {
		registered[c.Name] = true
	}
	for _, name := range txErrorCases {
		if !registered[name] {
			t.Errorf("%s case not registered on import", name)
		}
	}
}

func TestTxErrorCases_SkipForeignChain(t *testing.T) {
	ns, _ := attach.Build("wemix", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: txErrorCases})
	if len(rep.Results) != len(txErrorCases) {
		t.Fatalf("expected %d results, got %+v", len(txErrorCases), rep.Results)
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("expected skip on wemix for %s, got %s", r.Name, r.Status)
		}
	}
}
