package accounts_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the case

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// TestContractRoundtripCase_Registers confirms the case registers on import.
// The deployment itself is exercised by the real-chain E2E, not a mock.
func TestContractRoundtripCase_Registers(t *testing.T) {
	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "contract-roundtrip" {
			found = true
		}
	}
	if !found {
		t.Fatal("contract-roundtrip case not registered on import")
	}
}

// TestContractRoundtripCase_SkipsForeignChain confirms chain_compat gating.
func TestContractRoundtripCase_SkipsForeignChain(t *testing.T) {
	ns, _ := attach.Build("wemix", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"contract-roundtrip"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on foreign chain, got %+v", rep.Results)
	}
}
