package anzeon_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

var wsSubscribeCases = []string{
	"ws-subscribe-new-heads",
	"ws-subscribe-logs",
}

func TestWsSubscribeCases_Register(t *testing.T) {
	have := map[string]bool{}
	for _, c := range testkit.Cases() {
		have[c.Name] = true
	}
	for _, name := range wsSubscribeCases {
		if !have[name] {
			t.Errorf("case %q not registered", name)
		}
	}
}

// A pure-attach network exposes only "rpc" (no known WS port), so the WS cases
// must skip.
func TestWsSubscribeCases_SkipWithoutWSCap(t *testing.T) {
	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: wsSubscribeCases})
	if len(rep.Results) != len(wsSubscribeCases) {
		t.Fatalf("ran %d, want %d", len(rep.Results), len(wsSubscribeCases))
	}
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip without ws cap", r.Name, r.Status)
		}
	}
}
