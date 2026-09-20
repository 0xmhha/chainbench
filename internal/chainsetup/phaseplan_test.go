package chainsetup

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// TestPhasePlan_EachNodeCarriesItsOwnBinary.
//
// The bring-up actions a chain's start phase names (deploy governance, etcd
// init) attach to a node over its IPC socket, and the socket's path is derived
// from the binary. The plan handed to them gave every node the network's single
// binary, so in a network running more than one — which is what a swap case and
// a handoff across a fork are — the action waited on a socket the node never
// creates.
//
// The poa executor already prefers the plan's own entry for the node it runs
// on. This is the half that was filling the plan wrong.
func TestPhasePlan_EachNodeCarriesItsOwnBinary(t *testing.T) {
	w := &Workspace{state: State{
		Chain:    "stablenet",
		Target:   resource.Spec{DataRoot: "/data"},
		Binaries: map[string]string{"upgrade": "/data/bin/gstable-next"},
		Nodes: []node.Record{
			{Index: 1, DataDir: "/data/node1"},
			{Index: 2, DataDir: "/data/node2", Binary: "upgrade"},
		},
	}}

	plan := w.phasePlan("/data/bin/gstable")

	want := map[int]string{1: "/data/bin/gstable", 2: "/data/bin/gstable-next"}
	for _, spec := range plan.Nodes {
		if got := spec.Binary; got != want[spec.Index] {
			t.Errorf("node%d binary = %q, want %q", spec.Index, got, want[spec.Index])
		}
	}
	if len(plan.Nodes) != 2 {
		t.Fatalf("plan has %d nodes, want 2", len(plan.Nodes))
	}
}
