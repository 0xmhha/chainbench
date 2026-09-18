package blueprint_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/blueprint"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// topo is a layout the older format can express.
func topo(entries ...node.Entry) node.Topology {
	return node.Topology{Chain: "wbft", Nodes: entries}
}

// TestFromTopology_KeepsTheLayout is N6's absorption: the wider document has to
// describe the same network, node for node, or converting one is a migration
// that changes what runs.
func TestFromTopology_KeepsTheLayout(t *testing.T) {
	in := topo(
		node.Entry{Index: 1, Role: "bp"},
		node.Entry{Index: 3, Role: "endpoint", SyncMode: "archive"}, // out of order, legacy spelling
		node.Entry{Index: 2, Role: "bp", SyncMode: "full"},
	)
	bp, err := blueprint.FromTopology(in, blueprint.FromTopologyIn{Binary: "/opt/gwbft"})
	if err != nil {
		t.Fatalf("from topology: %v", err)
	}
	if bp.Chain != "wbft" {
		t.Errorf("chain = %q", bp.Chain)
	}
	// Sorted by index, which is launch order — the order the older format's
	// consumers already read it in.
	want := []struct{ name, role, sync string }{
		{"bp1", "bp", ""},
		{"bp2", "bp", "full"},
		{"en1", "en", "archive"},
	}
	if len(bp.Nodes) != len(want) {
		t.Fatalf("converted %d nodes, want %d", len(bp.Nodes), len(want))
	}
	for i, w := range want {
		got := bp.Nodes[i]
		if got.Name != w.name || got.Role != w.role || got.SyncMode != w.sync {
			t.Errorf("node %d = %s/%s/%q, want %s/%s/%q", i+1, got.Name, got.Role, got.SyncMode, w.name, w.role, w.sync)
		}
	}
	// The legacy spelling does not survive into the new document: one
	// vocabulary, decided at NM6.
	raw, err := blueprint.Marshal(bp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), "endpoint") {
		t.Errorf("the converted document still carries the legacy spelling:\n%s", raw)
	}
}

// TestFromTopology_ComposesTheSameNodeTable: the two documents must produce the
// same roles and modes through the composition, not merely look alike.
func TestFromTopology_ComposesTheSameNodeTable(t *testing.T) {
	in := topo(
		node.Entry{Index: 1, Role: "bp"},
		node.Entry{Index: 2, Role: "bp"},
		node.Entry{Index: 3, Role: "en", SyncMode: "snap"},
	)
	bp, err := blueprint.FromTopology(in, blueprint.FromTopologyIn{})
	if err != nil {
		t.Fatalf("from topology: %v", err)
	}
	for i, e := range in.Sorted() {
		if got, want := node.Role(bp.Nodes[i].Role), e.NodeRole(); got != want {
			t.Errorf("node %d role = %q, the topology says %q", i+1, got, want)
		}
		// A validator is pinned to full by the composition either way, so the
		// comparison is of what each document ASKS for.
		if got, want := bp.Nodes[i].SyncMode, e.SyncMode; got != want {
			t.Errorf("node %d sync = %q, the topology says %q", i+1, got, want)
		}
	}
}

// TestFromTopology_RefusesWhatItCannotCarry: a field with nowhere to land is
// reported, never dropped. A converted document that composed a different
// network from the file it came from is the failure this track exists to end.
func TestFromTopology_RefusesWhatItCannotCarry(t *testing.T) {
	for name, c := range map[string]struct {
		in    node.Topology
		wants string
	}{
		"the bootnode flag": {
			topo(node.Entry{Index: 1, Role: "bp", Bootnode: true}), "bootnode"},
		"a topology binary name": {
			topo(node.Entry{Index: 1, Role: "bp", Binary: "wbft"}), "binaries override"},
		"an empty layout": {topo(), "no nodes"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := blueprint.FromTopology(c.in, blueprint.FromTopologyIn{})
			if err == nil {
				t.Fatal("converted anyway")
			}
			if !strings.Contains(err.Error(), c.wants) {
				t.Errorf("error %q does not say %q", err, c.wants)
			}
		})
	}
}
