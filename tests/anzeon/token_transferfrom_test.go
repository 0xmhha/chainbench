package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the case

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func TestTokenTransferFromCase_Registers(t *testing.T) {
	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "token-transfer-from-moves-balance" {
			found = true
		}
	}
	if !found {
		t.Fatal("token-transfer-from-moves-balance case not registered")
	}
}

func TestTokenTransferFromCase_SkipsForeignChain(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"token-transfer-from-moves-balance"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wbft, got %+v", rep.Results)
	}
}
