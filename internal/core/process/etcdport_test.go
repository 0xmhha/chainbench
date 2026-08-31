package process_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// TestEtcdPortSurvivesIntoThePlan follows the one port that used to disappear.
//
// The wemix binary derives its etcd port as p2p+1, which is why the port plan
// insists on a p2p step of at least two. The plan knew that port and the
// runtime types did not, so the rule existed to protect a value that could not
// be asked for once a network was running: three representations of a node's
// ports, two of which dropped it in the conversion. This walks it from the
// assignment through plan assembly and asserts it is still there.
func TestEtcdPortSurvivesIntoThePlan(t *testing.T) {
	m, err := resource.Assign(resource.Pool{
		Hosts: []resource.Host{{Name: "local", Addr: "127.0.0.1"}},
		Slots: 4,
		Ports: resource.Bands{P2P: resource.Band{Base: 31000, Step: 10}, RPC: resource.Band{Base: 8600, Step: 10}},
	}, []resource.Request{{Role: node.RoleBP}, {Role: node.RoleBP}})
	if err != nil {
		t.Fatalf("Assign: %v", err)
	}

	var reqs []node.LaunchReq
	places := m.Placements()
	for _, p := range places {
		if p.Ports.Etcd != p.Ports.P2P+1 {
			t.Fatalf("%s: assignment lost the etcd port: %+v", p.Label, p.Ports)
		}
		reqs = append(reqs, node.LaunchReq{Role: p.Role, Binary: "gwemix"})
	}

	plan, err := process.PlanOf(launcherTestPlugin(), reqs, places, []byte(`{"g":1}`), "/d", nil)
	if err != nil {
		t.Fatalf("PlanOf: %v", err)
	}
	for _, spec := range plan.Nodes {
		if spec.Ports.Etcd != spec.Ports.P2P+1 {
			t.Fatalf("node%d: the plan lost the etcd port: %+v", spec.Index, spec.Ports)
		}
	}
	// And the derived port is inside the step, which is what the rule buys:
	// node2's p2p must not land on node1's etcd.
	if plan.Nodes[1].Ports.P2P == plan.Nodes[0].Ports.Etcd {
		t.Fatalf("node2 p2p collides with node1 etcd: %d", plan.Nodes[1].Ports.P2P)
	}
}
