package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

var masterMinterMemberCases = []string{"masterminter-member-add-remove"}

func TestMasterMinterMemberCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range masterMinterMemberCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

func TestMasterMinterMemberCases_SkipForeignChain(t *testing.T) {
	ns, _ := attach.Build("wemix", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: masterMinterMemberCases})
	if len(rep.Results) != len(masterMinterMemberCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(masterMinterMemberCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on wemix", r.Name, r.Status)
		}
	}
}
