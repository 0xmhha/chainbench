package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

var blacklistEnforcementCases = []string{
	"sender-blacklisted-rejected",
	"recipient-blacklisted-rejected",
	"unblacklist-restores",
	"effective-gas-price-regular",
}

func TestBlacklistEnforcementCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range blacklistEnforcementCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

func TestBlacklistEnforcementCases_SkipForeignChain(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: blacklistEnforcementCases})
	if len(rep.Results) != len(blacklistEnforcementCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(blacklistEnforcementCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on wbft", r.Name, r.Status)
		}
	}
}
