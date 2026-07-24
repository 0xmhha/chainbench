package hardfork_test

import (
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/pkg/chains/all"
	"github.com/0xmhha/chainbench/pkg/core/hardfork"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/registry"
)

func nodeSet(chain string, n int) node.NodeSet {
	ns := node.NodeSet{Chain: chain, Network: "local"}
	for i := 1; i <= n; i++ {
		ns.Nodes = append(ns.Nodes, node.Node{Index: i, Role: node.RoleValidator})
	}
	return ns
}

// TestBuildPlan_WemixToWbft reproduces the wemix-upgrade case: gwemix (poa) ->
// gwbft (wbft) at a fork block, keeping node datadirs.
func TestBuildPlan_WemixToWbft(t *testing.T) {
	from, _ := registry.Get("wemix")
	to, _ := registry.Get("wbft")
	ns := nodeSet("wemix", 3)

	plan, err := hardfork.BuildPlan(ns, from, to, 100, "/data")
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if plan.FromChain != "wemix" || plan.ToChain != "wbft" {
		t.Errorf("chains: %s -> %s", plan.FromChain, plan.ToChain)
	}
	if plan.FromBinary != "gwemix" || plan.ToBinary != "gwbft" {
		t.Errorf("binaries: %s -> %s", plan.FromBinary, plan.ToBinary)
	}
	if plan.Block != 100 || len(plan.Swaps) != 3 {
		t.Errorf("block=%d swaps=%d", plan.Block, len(plan.Swaps))
	}
	if !strings.HasSuffix(plan.Swaps[0].DataDir, "/data/node1") {
		t.Errorf("datadir: %s", plan.Swaps[0].DataDir)
	}
}

func TestBuildPlan_Errors(t *testing.T) {
	sn, _ := registry.Get("stablenet")
	wb, _ := registry.Get("wbft")

	if _, err := hardfork.BuildPlan(node.NodeSet{Chain: "stablenet"}, sn, wb, 0, "/d"); err == nil {
		t.Error("expected error for empty node set")
	}
	if _, err := hardfork.BuildPlan(nodeSet("stablenet", 1), sn, sn, 0, "/d"); err == nil {
		t.Error("expected error when from and to share a binary (gstable)")
	}
}
