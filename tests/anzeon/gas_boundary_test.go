package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

var gasBoundaryCases = []string{
	"feecap-exact-min-accepted",
	"feecap-above-min-accepted",
	"feecap-below-min-rejected",
	"legacy-gasprice-below-min-rejected",
	"accesslist-gasprice-below-min-rejected",
}

func TestGasBoundaryCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range gasBoundaryCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

func TestGasBoundaryCases_SkipForeignChain(t *testing.T) {
	ns, _ := node.AttachedSet("wbft", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: gasBoundaryCases})
	if len(rep.Results) != len(gasBoundaryCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(gasBoundaryCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on wbft", r.Name, r.Status)
		}
	}
}
