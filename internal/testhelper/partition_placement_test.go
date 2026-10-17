package testhelper

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestFirewallPartitionRejectsHostConflictsBeforeWrites(t *testing.T) {
	for _, tc := range []struct {
		name   string
		groups [][]node.Node
	}{
		{"late-peer", [][]node.Node{{nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.2", 30301)}, {nodeAt(3, "10.0.0.3", 30301), nodeAt(4, "10.0.0.1", 30401)}}},
		{"late-self", [][]node.Node{{nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.2", 30301)}, {nodeAt(3, "10.0.0.3", 30301), nodeAt(4, "10.0.0.2", 30401)}}},
		{"third-group", [][]node.Node{{nodeAt(1, "10.0.0.1", 30301)}, {nodeAt(2, "10.0.0.2", 30301)}, {nodeAt(3, "10.0.0.1", 30401)}}},
		{"repeated-node", [][]node.Node{{nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.2", 30301)}, {nodeAt(3, "10.0.0.3", 30301), nodeAt(1, "10.0.0.1", 30301)}}},
	} {
		for _, operation := range []struct {
			name   string
			insert bool
		}{{"cut", true}, {"restore", false}} {
			t.Run(tc.name+"/"+operation.name, func(t *testing.T) {
				h := &fakeHost{}
				err := fwApply(t.Context(), h, tc.groups, operation.insert)
				if err == nil || !strings.Contains(err.Error(), "same machine") {
					t.Fatalf("want a shared-host refusal, got %v", err)
				}
				if len(h.ran) != 0 {
					t.Fatalf("placement must be fully validated before firewall writes; ran %d command(s)", len(h.ran))
				}
			})
		}
	}
}

func TestFirewallPartitionAllowsSharedHostWithinOneGroup(t *testing.T) {
	for _, insert := range []bool{false, true} {
		h := &fakeHost{}
		groups := [][]node.Node{{nodeAt(1, "10.0.0.1", 30301), nodeAt(2, "10.0.0.1", 30401)}, {nodeAt(3, "10.0.0.3", 30301)}}
		if err := fwApply(t.Context(), h, groups, insert); err != nil {
			t.Fatalf("nodes on one host may stay in the same group: %v", err)
		}
		if len(h.ran) != 32 {
			t.Fatalf("want all four cross-group pairs, got %d commands", len(h.ran))
		}
	}
}
