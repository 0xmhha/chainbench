package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

var proposalExpiryCases = []string{"proposal-expiry-transitions"}

func TestProposalExpiryCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range proposalExpiryCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

// Without the short-expiry capability (a normal network) the case must skip — the
// default 7-day expiry cannot be observed in a test.
func TestProposalExpiryCases_SkipWithoutCap(t *testing.T) {
	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: proposalExpiryCases})
	if len(rep.Results) != len(proposalExpiryCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(proposalExpiryCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip without short-expiry cap", r.Name, r.Status)
		}
	}
}
