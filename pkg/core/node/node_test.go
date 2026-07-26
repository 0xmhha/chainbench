package node

import "testing"

func TestOffset(t *testing.T) {
	base := Endpoints{P2P: 30301, HTTP: 8501, WS: 9501, Auth: 8551, Metrics: 6061}
	// i=0 is identity.
	if got := Offset(base, 0); got != base {
		t.Errorf("Offset(base,0) = %+v, want %+v", got, base)
	}
	// i=2 advances every port by 2 (co-located node allocation, requirement #6).
	got := Offset(base, 2)
	want := Endpoints{P2P: 30303, HTTP: 8503, WS: 9503, Auth: 8553, Metrics: 6063}
	if got != want {
		t.Errorf("Offset(base,2) = %+v, want %+v", got, want)
	}
}

func TestPrimary(t *testing.T) {
	// Empty set -> not found.
	if _, ok := (NodeSet{}).Primary(); ok {
		t.Error("empty NodeSet should have no primary")
	}
	// Lowest Index wins, regardless of slice order.
	ns := NodeSet{Nodes: []Node{
		{Index: 3, RPCURL: "c"},
		{Index: 1, RPCURL: "a"},
		{Index: 2, RPCURL: "b"},
	}}
	p, ok := ns.Primary()
	if !ok || p.Index != 1 || p.RPCURL != "a" {
		t.Errorf("Primary() = %+v,%v want index 1 (a)", p, ok)
	}
}

func TestHasCapability(t *testing.T) {
	ns := NodeSet{Capabilities: []string{"rpc", "consensus"}}
	if !ns.HasCapability("rpc") || !ns.HasCapability("consensus") {
		t.Error("expected rpc and consensus capabilities")
	}
	if ns.HasCapability("ws") {
		t.Error("ws capability should be absent")
	}
	if (NodeSet{}).HasCapability("rpc") {
		t.Error("empty capability set should report false")
	}
}
