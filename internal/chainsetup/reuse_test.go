package chainsetup

import (
	"reflect"
	"sort"
	"testing"
)

// aliveAll marks every index in the table as answering.
func aliveAll(after []nodeTarget) map[int]bool {
	m := map[int]bool{}
	for _, t := range after {
		m[t.Index] = true
	}
	return m
}

func TestPlanReuse_AllMatchAndRunning(t *testing.T) {
	before := map[int]nodeBaseline{
		1: {Index: 1, ConfigHash: "a", Binary: "/gwbft", PID: 100},
		2: {Index: 2, ConfigHash: "b", Binary: "/gwbft", PID: 101},
	}
	after := []nodeTarget{
		{Index: 1, ConfigHash: "a", Binary: "/gwbft"},
		{Index: 2, ConfigHash: "b", Binary: "/gwbft"},
	}
	p := planReuse("g", "g", before, after, aliveAll(after))
	if p.Refuse != "" {
		t.Fatalf("unexpected refuse: %q", p.Refuse)
	}
	if got := p.redo(); len(got) != 0 {
		t.Fatalf("redo = %v, want none", got)
	}
	if p.reused() != 2 {
		t.Fatalf("reused = %d, want 2", p.reused())
	}
}

// TestPlanReuse_OneNodeDrifts is the case the user named: of many nodes, only
// the one whose config changed is redone; the rest are left running.
func TestPlanReuse_OneNodeDrifts(t *testing.T) {
	before := map[int]nodeBaseline{}
	after := make([]nodeTarget, 0, 15)
	for i := 1; i <= 15; i++ {
		before[i] = nodeBaseline{Index: i, ConfigHash: "cfg", Binary: "/gwbft", PID: 100 + i}
		after = append(after, nodeTarget{Index: i, ConfigHash: "cfg", Binary: "/gwbft"})
	}
	// Node 7's config drifted.
	after[6].ConfigHash = "cfg-new"

	p := planReuse("g", "g", before, after, aliveAll(after))
	if p.Refuse != "" {
		t.Fatalf("unexpected refuse: %q", p.Refuse)
	}
	if got := p.redo(); !reflect.DeepEqual(got, []int{7}) {
		t.Fatalf("redo = %v, want [7]", got)
	}
	if p.reused() != 14 {
		t.Fatalf("reused = %d, want 14", p.reused())
	}
}

func TestPlanReuse_BinaryChangeRedoesNode(t *testing.T) {
	before := map[int]nodeBaseline{1: {Index: 1, ConfigHash: "a", Binary: "/old", PID: 100}}
	after := []nodeTarget{{Index: 1, ConfigHash: "a", Binary: "/new"}}
	p := planReuse("g", "g", before, after, aliveAll(after))
	if got := p.redo(); !reflect.DeepEqual(got, []int{1}) {
		t.Fatalf("redo = %v, want [1]", got)
	}
	if p.Nodes[0].Reason != "binary changed" {
		t.Fatalf("reason = %q, want binary changed", p.Nodes[0].Reason)
	}
}

func TestPlanReuse_DeadOrSilentNodeRedone(t *testing.T) {
	before := map[int]nodeBaseline{
		1: {Index: 1, ConfigHash: "a", Binary: "/b", PID: 0},   // never ran
		2: {Index: 2, ConfigHash: "a", Binary: "/b", PID: 200}, // ran, but silent
	}
	after := []nodeTarget{
		{Index: 1, ConfigHash: "a", Binary: "/b"},
		{Index: 2, ConfigHash: "a", Binary: "/b"},
	}
	alive := map[int]bool{1: false, 2: false}
	p := planReuse("g", "g", before, after, alive)
	got := p.redo()
	sort.Ints(got)
	if !reflect.DeepEqual(got, []int{1, 2}) {
		t.Fatalf("redo = %v, want [1 2]", got)
	}
	if p.Nodes[0].Reason != "not running" {
		t.Fatalf("node1 reason = %q, want not running", p.Nodes[0].Reason)
	}
	if p.Nodes[1].Reason != "not answering" {
		t.Fatalf("node2 reason = %q, want not answering", p.Nodes[1].Reason)
	}
}

func TestPlanReuse_GenesisChangeRefusesWholeReuse(t *testing.T) {
	before := map[int]nodeBaseline{1: {Index: 1, ConfigHash: "a", Binary: "/b", PID: 100}}
	after := []nodeTarget{{Index: 1, ConfigHash: "a", Binary: "/b"}}
	p := planReuse("g-old", "g-new", before, after, aliveAll(after))
	if p.Refuse == "" {
		t.Fatal("expected the whole reuse to be refused on a genesis change")
	}
	// A refusal touches no node.
	if len(p.redo()) != 0 {
		t.Fatalf("a refusal must plan no redo, got %v", p.redo())
	}
}

func TestReusePlan_Describe(t *testing.T) {
	cases := []struct {
		name string
		plan reusePlan
		want string
	}{
		{"refuse", reusePlan{Refuse: "genesis changed"}, "reuse refused: genesis changed"},
		{"all-match", reusePlan{Nodes: []reuseDisposition{{Index: 1, Reuse: true}, {Index: 2, Reuse: true}}},
			"reuse-if-matching: all 2 node(s) match and are running; nothing to redo"},
		{"partial", reusePlan{Nodes: []reuseDisposition{{Index: 1, Reuse: true}, {Index: 2, Reason: "config changed"}}},
			"reuse-if-matching: 1 node(s) reused, 1 redone [2]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.plan.describe(); got != tc.want {
				t.Fatalf("describe() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestPlanReuse_FirstUpHasNoBaseline: no prior genesis and no prior nodes means
// a first up — every node is composed fresh, and it is not a genesis-change
// refusal.
func TestPlanReuse_FirstUpHasNoBaseline(t *testing.T) {
	after := []nodeTarget{{Index: 1, ConfigHash: "a", Binary: "/b"}}
	p := planReuse("", "g", map[int]nodeBaseline{}, after, aliveAll(after))
	if p.Refuse != "" {
		t.Fatalf("first up must not refuse: %q", p.Refuse)
	}
	if got := p.redo(); !reflect.DeepEqual(got, []int{1}) {
		t.Fatalf("redo = %v, want [1]", got)
	}
	if p.Nodes[0].Reason != "not running" {
		t.Fatalf("reason = %q, want not running", p.Nodes[0].Reason)
	}
}
