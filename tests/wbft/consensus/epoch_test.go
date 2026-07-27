package consensus_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/consensus" // register the cases

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

var epochCases = []string{"epoch-transition-carries-epoch-info"}

func TestEpochCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range epochCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

func TestEpochCases_SkipForeignChain(t *testing.T) {
	ns, _ := attach.Build("wemix", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: epochCases})
	if len(rep.Results) != len(epochCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(epochCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on wemix", r.Name, r.Status)
		}
	}
}
