package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

var forkTransitionCases = []string{
	"govminter-code-changes-at-boho",
	"p256-inactive-before-boho",
	"anzeon-active-before-boho",
	"prealloc-preserved-across-boho",
}

func TestForkTransitionCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range forkTransitionCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

// A normal stablenet network activates Boho at genesis and exposes only "rpc",
// so the fork-transition cases (which require the "delayed-boho" capability)
// must skip — they only apply to a network launched with a delayed fork.
func TestForkTransitionCases_SkipWithoutDelayedCap(t *testing.T) {
	ns, _ := node.AttachedSet("stablenet", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: forkTransitionCases})
	if len(rep.Results) != len(forkTransitionCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(forkTransitionCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip without delayed-boho cap", r.Name, r.Status)
		}
	}
}
