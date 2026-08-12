package accounts_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the case

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func TestContractEventCase_Registers(t *testing.T) {
	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "contract-event-emitted" {
			found = true
		}
	}
	if !found {
		t.Fatal("contract-event-emitted case not registered")
	}
}

func TestContractEventCase_SkipsForeignChain(t *testing.T) {
	ns, _ := node.AttachedSet("wemix", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"contract-event-emitted"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wemix, got %+v", rep.Results)
	}
}
