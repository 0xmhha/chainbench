package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

var govAccessControlCases = []string{
	"direct-blacklist-call-rejected",
	"non-member-configure-minter-rejected",
}

func TestGovAccessControlCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range govAccessControlCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

func TestGovAccessControlCases_SkipForeignChain(t *testing.T) {
	ns, _ := node.AttachedSet("wemix", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: govAccessControlCases})
	if len(rep.Results) != len(govAccessControlCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(govAccessControlCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on wemix", r.Name, r.Status)
		}
	}
}
