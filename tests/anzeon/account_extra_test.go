package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

var accountExtraCases = []string{
	"authorized-extra-bit-synced",
	"blacklisted-extra-bit-synced",
	"dual-status-extra",
	"extra-balance-preserved",
}

func TestAccountExtraCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range accountExtraCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

// A normal stablenet network exposes only "rpc"; the account-extra cases require
// the "account-extra" capability (advertised only when launched with the
// account-extra overlay), so they must skip here.
func TestAccountExtraCases_SkipWithoutCap(t *testing.T) {
	ns, _ := node.AttachedSet("stablenet", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: accountExtraCases})
	if len(rep.Results) != len(accountExtraCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(accountExtraCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip without account-extra cap", r.Name, r.Status)
		}
	}
}
