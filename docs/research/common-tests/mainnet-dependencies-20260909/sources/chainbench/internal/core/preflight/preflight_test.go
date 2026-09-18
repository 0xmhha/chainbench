package preflight_test

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/preflight"
)

func have() preflight.Have {
	return preflight.Have{
		Chain: "stablenet", Binary: "/bin/gstable", KeysDir: "keys/preset", Validators: 2, Started: true,
		Nodes: []preflight.Node{
			{Index: 1, Role: node.RoleBP, Host: "127.0.0.1", PID: 100},
			{Index: 2, Role: node.RoleBP, Host: "127.0.0.1", PID: 101},
			{Index: 3, Role: node.RoleEN, SyncMode: "full", Host: "127.0.0.1", PID: 102},
		},
	}
}

func TestCompare(t *testing.T) {
	for _, tc := range []struct {
		name   string
		have   preflight.Have
		want   preflight.Want
		expect preflight.Verdict
		nodes  []int
		reason string
	}{
		{"same shape reuses", have(), preflight.Want{Chain: "stablenet", Validators: 2, Endpoints: 1}, preflight.Reuse, nil, ""},
		{"nothing composed", preflight.Have{}, preflight.Want{Chain: "stablenet"}, preflight.Compose, nil, "nothing is composed"},
		{"other chain rebuilds all", have(), preflight.Want{Chain: "wbft", Validators: 2, Endpoints: 1}, preflight.RebuildAll, nil, "chain:"},
		{"more validators rebuilds all", have(), preflight.Want{Chain: "stablenet", Validators: 3, Endpoints: 1}, preflight.RebuildAll, nil, "validators:"},
		{"other keys rebuild all", have(), preflight.Want{Chain: "stablenet", KeysDir: "keys/other"}, preflight.RebuildAll, nil, "keys:"},
		{"never started rebuilds all", func() preflight.Have { h := have(); h.Started = false; return h }(), preflight.Want{Chain: "stablenet"}, preflight.RebuildAll, nil, "never started"},
		{"one node's sync mode rebuilds that node", have(),
			preflight.Want{Chain: "stablenet", Nodes: []preflight.Node{{Index: 3, SyncMode: "snap"}}},
			preflight.RebuildNodes, []int{3}, `node3: sync "full"→"snap"`},
		{"a node moved to another server rebuilds that node", have(),
			preflight.Want{Chain: "stablenet", Nodes: []preflight.Node{{Index: 2, Server: "server5"}}},
			preflight.RebuildNodes, []int{2}, "server"},
		{"a wanted node that is not composed rebuilds all", have(),
			preflight.Want{Chain: "stablenet", Nodes: []preflight.Node{{Index: 9, SyncMode: "snap"}}},
			preflight.RebuildAll, nil, "node9 is wanted"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := preflight.Compare(tc.have, tc.want)
			if d.Verdict != tc.expect {
				t.Fatalf("verdict = %s, want %s (%s)", d.Verdict, tc.expect, d)
			}
			if len(tc.nodes) > 0 && strings.Join(itoa(d.Nodes), ",") != strings.Join(itoa(tc.nodes), ",") {
				t.Fatalf("nodes = %v, want %v", d.Nodes, tc.nodes)
			}
			if tc.reason != "" && !strings.Contains(d.String(), tc.reason) {
				t.Fatalf("reasons should mention %q: %s", tc.reason, d)
			}
		})
	}
}

// TestCheck_ADeadNodeJoinsTheRebuild: the paper comparison keeps every node;
// the live probe finds node2 gone, so node2 alone is rebuilt.
func TestCheck_ADeadNodeJoinsTheRebuild(t *testing.T) {
	live := func(_ context.Context, n preflight.Node) (bool, string) {
		if n.Index == 2 {
			return false, "pid 101 not running"
		}
		return true, ""
	}
	d := preflight.Check(context.Background(), have(), preflight.Want{Chain: "stablenet", Validators: 2, Endpoints: 1}, live)
	if d.Verdict != preflight.RebuildNodes || len(d.Nodes) != 1 || d.Nodes[0] != 2 {
		t.Fatalf("decision = %s", d)
	}
	if !strings.Contains(d.String(), "pid 101 not running") {
		t.Fatalf("the probe's reason should survive: %s", d)
	}
}

// TestCheck_EveryNodeDeadIsNotARestart: a network with nothing alive comes
// back whole through the composition, not node by node.
func TestCheck_EveryNodeDeadIsNotARestart(t *testing.T) {
	dead := func(context.Context, preflight.Node) (bool, string) { return false, "" }
	d := preflight.Check(context.Background(), have(), preflight.Want{Chain: "stablenet"}, dead)
	if d.Verdict != preflight.RebuildAll {
		t.Fatalf("decision = %s, want rebuild-all", d)
	}
}

// TestCheck_NetworkWideVerdictIsNotProbed: when the chain has to be recomposed
// anyway, no node is dialled.
func TestCheck_NetworkWideVerdictIsNotProbed(t *testing.T) {
	probed := 0
	live := func(context.Context, preflight.Node) (bool, string) { probed++; return true, "" }
	d := preflight.Check(context.Background(), have(), preflight.Want{Chain: "wbft"}, live)
	if d.Verdict != preflight.RebuildAll || probed != 0 {
		t.Fatalf("decision = %s, probed %d nodes", d, probed)
	}
}

func itoa(in []int) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = string(rune('0' + v))
	}
	return out
}
