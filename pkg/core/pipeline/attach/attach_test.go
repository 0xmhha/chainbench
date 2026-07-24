package attach

import (
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/node"
)

func TestBuild(t *testing.T) {
	ns, err := Build("wbft", "remote-net", []Endpoint{
		{RPCURL: "http://a:8501", Host: "a", HTTPPort: 8501},
		{RPCURL: "http://b:8501", Host: "b", HTTPPort: 8501},
	})
	if err != nil {
		t.Fatal(err)
	}
	if ns.Chain != "wbft" || ns.Network != "remote-net" {
		t.Errorf("identity: %s/%s", ns.Chain, ns.Network)
	}
	if len(ns.Nodes) != 2 {
		t.Fatalf("nodes: %d", len(ns.Nodes))
	}
	if ns.Nodes[0].Index != 1 || ns.Nodes[0].RPCURL != "http://a:8501" || ns.Nodes[0].Role != node.RoleEndpoint {
		t.Errorf("node1: %+v", ns.Nodes[0])
	}
	if p, ok := ns.Primary(); !ok || p.Index != 1 {
		t.Errorf("primary: %+v ok=%v", p, ok)
	}
	if !ns.HasCapability("rpc") {
		t.Error("attached set should have rpc capability")
	}
}

func TestBuild_Errors(t *testing.T) {
	if _, err := Build("wbft", "n", nil); err == nil {
		t.Error("expected error for no endpoints")
	}
	if _, err := Build("wbft", "n", []Endpoint{{RPCURL: ""}}); err == nil {
		t.Error("expected error for empty rpc url")
	}
}
