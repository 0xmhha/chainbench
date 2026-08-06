package accounts_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the case

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// TestValueTransferCase_Registers confirms the case registers on import. The
// transaction behavior itself is exercised by the real-chain E2E, not a mock
// (it drives the accounts SDK signing/submit path against a live node).
func TestValueTransferCase_Registers(t *testing.T) {
	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "value-transfer" {
			found = true
		}
	}
	if !found {
		t.Fatal("value-transfer case not registered on import")
	}
}

// TestValueTransferCase_SkipsForeignChain confirms chain_compat gating: the
// case does not run outside its ChainCompat (the wbft family), e.g. on wemix.
func TestValueTransferCase_SkipsForeignChain(t *testing.T) {
	ns, _ := attach.Build("wemix", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"value-transfer"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on foreign chain, got %+v", rep.Results)
	}
}
