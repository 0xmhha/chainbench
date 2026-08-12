package accounts_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the case

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func TestSetCodeCase_Registers(t *testing.T) {
	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "set-code-delegation" {
			found = true
			if c.Category != "accounts" {
				t.Errorf("category = %q, want accounts", c.Category)
			}
		}
	}
	if !found {
		t.Fatal("set-code-delegation case not registered on import")
	}
}

func TestSetCodeCase_SkipsForeignChain(t *testing.T) {
	ns, _ := node.AttachedSet("wemix", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"set-code-delegation"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wemix, got %+v", rep.Results)
	}
}
