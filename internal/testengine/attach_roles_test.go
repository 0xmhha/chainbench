package testengine_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// A spec addresses a node two ways: "node1" is the first node by order, "en1"
// the first node whose role is en. Both readings need the table to say what
// each node is, and attach has two kinds of table.

// TestAttach_EndpointsAloneMakeEveryNodeAnEndpoint: given only URLs, no node
// may be assumed to produce, so every one is an endpoint. This is the correct
// answer to a question chainbench cannot otherwise answer.
func TestAttach_EndpointsAloneMakeEveryNodeAnEndpoint(t *testing.T) {
	build := testengine.NewAttachBuildEnv("stablenet", []node.RPCEndpoint{
		{RPCURL: "http://127.0.0.1:8600"},
		{RPCURL: "http://127.0.0.1:8610"},
	})
	ns, teardown, err := build(context.Background(), nil, dsl.Spec{})
	if err != nil {
		t.Fatalf("attach build: %v", err)
	}
	if teardown != nil {
		t.Error("attach must not offer to stop nodes it did not start")
	}
	for _, n := range ns.Nodes {
		if n.Role != node.RoleEndpoint {
			t.Errorf("node%d came back as %q; with no record, no node may be assumed to produce", n.Index, n.Role)
		}
	}
}

// TestAttach_ARecordedSetKeepsItsRoles: given the composer's record, the roles
// survive, so "en1" reaches an endpoint and not the first producer.
//
// Measured 2026-09-06: flattening the record to URLs made every node an
// endpoint, so "en1" resolved to node1 — a producer — and a spec that meant
// the endpoint got one silently.
func TestAttach_ARecordedSetKeepsItsRoles(t *testing.T) {
	recorded := node.NodeSet{
		Chain: "stablenet", Network: "local",
		Capabilities: []string{"rpc", "short-expiry"},
		Nodes: []node.Node{
			{Index: 1, Role: node.RoleValidator, RPCURL: "http://127.0.0.1:8600"},
			{Index: 2, Role: node.RoleValidator, RPCURL: "http://127.0.0.1:8610"},
			{Index: 5, Role: node.RoleEndpoint, RPCURL: "http://127.0.0.1:8640"},
		},
	}
	ns, teardown, err := testengine.NewRecordedBuildEnv(recorded)(context.Background(), nil, dsl.Spec{})
	if err != nil {
		t.Fatalf("recorded build: %v", err)
	}
	if teardown != nil {
		t.Error("this run did not start the nodes and must not offer to stop them")
	}
	var endpoints []int
	for _, n := range ns.Nodes {
		if node.Is(n.Role, node.RoleEN) {
			endpoints = append(endpoints, n.Index)
		}
	}
	if len(endpoints) != 1 || endpoints[0] != 5 {
		t.Errorf("the endpoints came back as %v; the record says node5 alone", endpoints)
	}
	if ns.Chain != "stablenet" {
		t.Errorf("the chain was lost: %q", ns.Chain)
	}
}
