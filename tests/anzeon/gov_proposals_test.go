package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// govWriteCases are the stablenet-only governance write cases ported from
// f-system-contracts; each must register and gate to stablenet.
var govWriteCases = []string{
	"burn-proposal-executes",
	"validator-add-member-executes",
	"blacklist-proposal-executes",
	"authorize-proposal-executes",
	"configure-minter-proposal-executes",
	"burn-cancel-refundable",
	"burn-execute-no-refundable",
	"claim-zero-refund-reverts",
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
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://x"}})
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
