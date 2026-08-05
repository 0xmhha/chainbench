package accounts_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the cases

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// txMetadataCases are the live post-transfer RPC-metadata cases in this file.
// Their transaction behavior is exercised by the real-chain E2E, not a mock
// (they drive the accounts SDK signing/submit path against a live node), so the
// sibling tests validate registration and chain gating only.
var txMetadataCases = []string{
	"transaction-count-increments",
	"transaction-by-hash-fields",
	"transaction-receipt-fields",
}

func TestTxMetadataCases_Register(t *testing.T) {
	registered := map[string]bool{}
	for _, c := range testkit.Cases() {
		registered[c.Name] = true
	}
	for _, name := range txMetadataCases {
		if !registered[name] {
			t.Errorf("%s case not registered on import", name)
		}
	}
}

func TestTxMetadataCases_SkipForeignChain(t *testing.T) {
	ns, _ := attach.Build("wemix", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: txMetadataCases})
	if len(rep.Results) != len(txMetadataCases) {
		t.Fatalf("expected %d results, got %+v", len(txMetadataCases), rep.Results)
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("expected skip on wemix for %s, got %s", r.Name, r.Status)
		}
	}
}
