package external_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/external" // register the cases

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

var externalCases = []string{
	"external-value-transfer",
	"external-fee-delegated-transfer",
}

func TestExternalCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range externalCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

// Without a funded key the chain-agnostic write cases must skip themselves — even
// though they are rpc-only and chain-agnostic, so they would otherwise "run".
// They skip before any RPC call, so no live endpoint is needed here.
func TestExternalCases_SkipWithoutFundedKey(t *testing.T) {
	ns, _ := attach.Build("layer2", "attached", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: externalCases}) // no FundedKey
	if len(rep.Results) != len(externalCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(externalCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip without a funded key", r.Name, r.Status)
		}
	}
}
