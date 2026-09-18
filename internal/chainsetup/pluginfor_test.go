package chainsetup

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register the chains a composition can name
)

// TestPluginFor_ANodeIsConfiguredAgainstItsOwnChain.
//
// A network can run two builds, and the chain is what answers a node's launch:
// which flag vocabulary its binary accepts, which RPC namespace it serves, and
// what its consensus asks of it. The composition had one plugin for every node,
// so a node running the other build assembled its argv and its config against
// the wrong build's answers — wemix's flag spellings for a go-wbft binary, or
// the other way round.
//
// It mirrors binaryFor and genesisFor on purpose. The three ask the same
// question — which of this network's builds is this node — and keeping them the
// same shape is what stops one of them answering differently.
func TestPluginFor_ANodeIsConfiguredAgainstItsOwnChain(t *testing.T) {
	w := &Workspace{state: State{
		Chain:        "wemix",
		Binaries:     map[string]string{"next": "/data/bin/gwbft"},
		BinaryChains: map[string]string{"next": "wbft"},
		Nodes: []node.Record{
			{Index: 1},
			{Index: 2, Binary: "next"},
			// A binary with no chain of its own runs the composition's:
			// declaring a second build is not declaring a second chain.
			{Index: 3, Binary: "other"},
		},
	}}

	want := map[int]struct{ chain, namespace, dialect string }{
		1: {"wemix", "wemix", "geth110-wemix"},
		2: {"wbft", "istanbul", "geth114"},
		3: {"wemix", "wemix", "geth110-wemix"},
	}
	for _, ns := range w.state.Nodes {
		p, err := w.pluginFor(ns)
		if err != nil {
			t.Fatalf("node%d: %v", ns.Index, err)
		}
		facts := nodeconfig.ChainOf(p, node.Role(ns.Role))
		got := struct{ chain, namespace, dialect string }{facts.ID, facts.RPCNamespace, facts.Dialect}
		if got != want[ns.Index] {
			t.Errorf("node%d = %+v, want %+v", ns.Index, got, want[ns.Index])
		}
	}
}

// TestPluginFor_ASwapKeepsTheChainWithTheName: a swap renames the entry to the
// node it belongs to, and the chain has to travel with it. Without that the
// node relaunches on the other build's flag vocabulary while running the binary
// it was swapped onto.
func TestPluginFor_ASwapKeepsTheChainWithTheName(t *testing.T) {
	w := &Workspace{state: State{
		Chain:        "wemix",
		Binaries:     map[string]string{"next": "/data/bin/gwbft"},
		BinaryChains: map[string]string{"next": "wbft"},
		Nodes:        []node.Record{{Index: 2}},
	}}
	w.setNodeBinary(0, "next")

	p, err := w.pluginFor(w.state.Nodes[0])
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Manifest().ID; got != "wbft" {
		t.Errorf("after the swap the node runs chain %q, want wbft", got)
	}
	if got := w.binaryFor(w.state.Nodes[0], "/data/bin/gwemix"); got != "/data/bin/gwbft" {
		t.Errorf("after the swap the node runs binary %q", got)
	}
}
