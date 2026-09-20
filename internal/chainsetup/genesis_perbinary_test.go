package chainsetup

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestGenesisFor_ANodeInitializesFromItsOwnBinarysGenesis.
//
// A network can run two builds that do not accept the same genesis: the
// successor across a fork needs fork settings its predecessor may refuse, and
// whether it refuses them is a fact about those two builds. Until this the
// network had one genesis document and every node was initialized from it.
func TestGenesisFor_ANodeInitializesFromItsOwnBinarysGenesis(t *testing.T) {
	w := &Workspace{state: State{
		GenesisPath:  "/data/genesis.json",
		GenesisPaths: map[string]string{"next": "/data/genesis-next.json"},
		Nodes: []node.Record{
			{Index: 1},
			{Index: 2, Binary: "next"},
			// A binary with no genesis of its own keeps the network's: declaring
			// a second binary is not declaring a second chain.
			{Index: 3, Binary: "other"},
		},
	}}
	want := map[int]string{
		1: "/data/genesis.json",
		2: "/data/genesis-next.json",
		3: "/data/genesis.json",
	}
	for _, ns := range w.state.Nodes {
		if got := w.genesisFor(ns); got != want[ns.Index] {
			t.Errorf("node%d genesis = %q, want %q", ns.Index, got, want[ns.Index])
		}
	}
}

// TestGenesisPaths_EveryDocumentOnce is what rm, the run record and the
// pre-launch check all walk. The network's comes first, and a name mapped to
// the network's own path is not listed twice.
func TestGenesisPaths_EveryDocumentOnce(t *testing.T) {
	w := &Workspace{state: State{
		GenesisPath: "/data/genesis.json",
		GenesisPaths: map[string]string{
			"next": "/data/genesis-next.json",
			"same": "/data/genesis.json",
		},
	}}
	got := w.genesisPaths()
	want := []string{"/data/genesis.json", "/data/genesis-next.json"}
	if len(got) != len(want) {
		t.Fatalf("genesisPaths = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("genesisPaths = %v, want %v", got, want)
		}
	}
}

// TestGenesisPaths_NoVariantsIsTheNetworkAlone keeps the ordinary composition
// exactly as it was: one document, listed once.
func TestGenesisPaths_NoVariantsIsTheNetworkAlone(t *testing.T) {
	w := &Workspace{state: State{GenesisPath: "/data/genesis.json"}}
	if got := w.genesisPaths(); len(got) != 1 || got[0] != "/data/genesis.json" {
		t.Fatalf("genesisPaths = %v", got)
	}
}
