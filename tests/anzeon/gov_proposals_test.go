package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// govWriteCases are the stablenet-only governance write cases still on the
// Go-func path (the rest are DSL specs under tests/specs/system-contracts);
// each must register and gate to stablenet.
var govWriteCases = []string{
	"validator-add-member-executes",
	"masterminter-member-add-remove",
}

func TestGovWriteCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range govWriteCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

func TestGovWriteCases_SkipForeignChain(t *testing.T) {
	ns, _ := node.AttachedSet("wbft", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: govWriteCases})
	if len(rep.Results) != len(govWriteCases) {
		t.Fatalf("ran %d cases, want %d", len(rep.Results), len(govWriteCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on wbft", r.Name, r.Status)
		}
	}
}
