package accounts_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the case

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// TestFeeDelegatedCase_Registers confirms the case registers on import. The 0x16
// transaction behavior is exercised by the real-chain E2E, not a mock.
func TestFeeDelegatedCase_Registers(t *testing.T) {
	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "fee-delegated-transfer" {
			found = true
			if c.Category != "accounts" {
				t.Errorf("category = %q, want accounts", c.Category)
			}
		}
	}
	if !found {
		t.Fatal("fee-delegated-transfer case not registered on import")
	}
}

// TestFeeDelegatedCase_SkipsForeignChain confirms chain_compat gating.
func TestFeeDelegatedCase_SkipsForeignChain(t *testing.T) {
	ns, _ := attach.Build("wemix", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"fee-delegated-transfer"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on foreign chain, got %+v", rep.Results)
	}
}
