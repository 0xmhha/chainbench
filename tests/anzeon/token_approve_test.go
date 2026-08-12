package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the case

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func TestTokenApproveCase_Registers(t *testing.T) {
	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "token-approve-sets-allowance" {
			found = true
		}
	}
	if !found {
		t.Fatal("token-approve-sets-allowance case not registered")
	}
}

func TestTokenApproveCase_SkipsForeignChain(t *testing.T) {
	ns, _ := node.AttachedSet("wbft", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"token-approve-sets-allowance"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wbft, got %+v", rep.Results)
	}
}
