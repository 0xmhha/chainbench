package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the case

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

func TestMintProposalCase_Registers(t *testing.T) {
	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "mint-proposal-executes" {
			found = true
		}
	}
	if !found {
		t.Fatal("mint-proposal-executes case not registered")
	}
}

func TestMintProposalCase_SkipsForeignChain(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"mint-proposal-executes"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wbft (stablenet-only case), got %+v", rep.Results)
	}
}
