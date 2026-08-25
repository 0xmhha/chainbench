package chainsetup_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/registry"

	_ "github.com/0xmhha/chainbench/internal/chains/wbft" // register the wbft plugin
)

func wbftPlugin(t *testing.T) registry.ChainPlugin {
	t.Helper()
	p, err := registry.Get("wbft")
	if err != nil {
		t.Fatalf("registry.Get(wbft): %v", err)
	}
	return p
}

func placed(role node.Role, host string, p2p, http int, dataPath string) chainsetup.PlacedNode {
	return chainsetup.PlacedNode{
		Req: place.NodeReq{Role: role, Binary: "go-wbft"},
		Placement: netmap.Placement{
			Host:    host,
			Ports:   netmap.Ports{P2P: p2p, Etcd: p2p + 1, HTTP: http, WS: http + 1, Auth: http + 2},
			DataDir: dataPath,
		},
	}
}

func TestAssemblePlan_UsesAllocatorPorts(t *testing.T) {
	nodes := []chainsetup.PlacedNode{
		placed(node.RoleValidator, "127.0.0.1", 30301, 8501, "/data/node1"),
		// Deliberately non-stepped ports to prove the plan takes them from the
		// placement, not from a fixed base+step like setup.BuildPlan.
		placed(node.RoleValidator, "127.0.0.1", 41000, 9000, "/data/node2"),
	}
	genesis := []byte(`{"config":{}}`)

	plan, err := chainsetup.AssemblePlan(wbftPlugin(t), nodes, genesis, "/data", []string{"ws"})
	if err != nil {
		t.Fatalf("AssemblePlan: %v", err)
	}
	if plan.Chain != "wbft" {
		t.Fatalf("Chain = %q, want wbft", plan.Chain)
	}
	if string(plan.Genesis) != string(genesis) {
		t.Fatalf("Genesis not carried through")
	}
	if len(plan.Nodes) != 2 {
		t.Fatalf("Nodes = %d, want 2", len(plan.Nodes))
	}

	n1 := plan.Nodes[0]
	if n1.Index != 1 || n1.Role != node.RoleValidator || n1.Host != "127.0.0.1" {
		t.Fatalf("node1 header wrong: %+v", n1)
	}
	if n1.Ports.HTTP != 8501 || n1.Ports.P2P != 30301 || n1.Ports.WS != 8502 || n1.Ports.Auth != 8503 {
		t.Fatalf("node1 ports wrong: %+v", n1.Ports)
	}
	if n1.DataDir != "/data/node1" {
		t.Fatalf("node1 DataDir = %q", n1.DataDir)
	}
	// Argv assembly is single-sited in the launcher's arming step (launchopt
	// Builder); the plan deliberately carries no Args.
	if len(n1.Args) != 0 {
		t.Fatalf("plan must not pre-assemble Args (moved to arming): %v", n1.Args)
	}

	// node2 must carry the placement's non-stepped ports, proving allocator-driven.
	if plan.Nodes[1].Ports.HTTP != 9000 || plan.Nodes[1].Ports.P2P != 41000 {
		t.Fatalf("node2 ports not from placement: %+v", plan.Nodes[1].Ports)
	}
}

func TestAssemblePlan_BinaryFallsBackToManifest(t *testing.T) {
	pn := chainsetup.PlacedNode{
		Req:       place.NodeReq{Role: node.RoleValidator}, // no Binary
		Placement: netmap.Placement{Host: "127.0.0.1", Ports: netmap.Ports{P2P: 30301, HTTP: 8501}, DataDir: "/d/node1"},
	}
	plan, err := chainsetup.AssemblePlan(wbftPlugin(t), []chainsetup.PlacedNode{pn}, nil, "/d", nil)
	if err != nil {
		t.Fatalf("AssemblePlan: %v", err)
	}
	if got := plan.Nodes[0].Binary; got != wbftPlugin(t).Manifest().Binary {
		t.Fatalf("binary = %q, want manifest binary %q", got, wbftPlugin(t).Manifest().Binary)
	}
}

func TestAssemblePlan_DataDirDefault(t *testing.T) {
	pn := chainsetup.PlacedNode{
		Req:       place.NodeReq{Role: node.RoleValidator, Binary: "go-wbft"},
		Placement: netmap.Placement{Host: "127.0.0.1", Ports: netmap.Ports{P2P: 30301, HTTP: 8501}}, // no DataPath
	}
	plan, err := chainsetup.AssemblePlan(wbftPlugin(t), []chainsetup.PlacedNode{pn}, nil, "/root", nil)
	if err != nil {
		t.Fatalf("AssemblePlan: %v", err)
	}
	if got := plan.Nodes[0].DataDir; got != "/root/node1" {
		t.Fatalf("default DataDir = %q, want /root/node1", got)
	}
}

func TestAssemblePlan_NoNodes(t *testing.T) {
	if _, err := chainsetup.AssemblePlan(wbftPlugin(t), nil, nil, "/d", nil); err == nil {
		t.Fatal("expected error for empty placements")
	}
}
