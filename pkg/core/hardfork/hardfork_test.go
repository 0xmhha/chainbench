package hardfork_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/pkg/chains/all"
	"github.com/0xmhha/chainbench/pkg/core/driver"
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

// TestExecute_StopsAndRelaunches stops a running "from" node and relaunches it
// on the target binary, verifying the swap end to end with fake processes.
func TestExecute_StopsAndRelaunches(t *testing.T) {
	old := exec.Command("sleep", "30")
	if err := old.Start(); err != nil {
		t.Fatal(err)
	}
	oldPID := old.Process.Pid

	from, _ := registry.Get("wemix")
	to, _ := registry.Get("wbft")
	dir := t.TempDir()
	ns := node.NodeSet{Chain: "wemix", Nodes: []node.Node{
		{Index: 1, Role: node.RoleValidator, Ports: node.Endpoints{HTTP: 8501, P2P: 30301}, PID: oldPID},
	}}
	plan, err := hardfork.BuildPlan(ns, from, to, 100, dir)
	if err != nil {
		t.Fatal(err)
	}

	// fake target binary: init exits 0; run sleeps.
	bin := filepath.Join(dir, "faketo")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nif [ \"$1\" = \"init\" ]; then exit 0; fi\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	newNS, err := plan.Execute(context.Background(), driver.NewLocalDriver(), to.Family(), bin)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	// The old node was stopped.
	if werr := old.Wait(); werr == nil {
		t.Error("old node should have been stopped")
	}
	// The new set is the target chain with a fresh PID.
	if newNS.Chain != "wbft" || len(newNS.Nodes) != 1 {
		t.Fatalf("new nodeset: chain=%s nodes=%d", newNS.Chain, len(newNS.Nodes))
	}
	np := newNS.Nodes[0].PID
	if np == 0 || np == oldPID {
		t.Errorf("new pid %d (old %d)", np, oldPID)
	}
	// cleanup the relaunched fake process.
	if p, err := os.FindProcess(np); err == nil {
		_ = p.Kill()
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
