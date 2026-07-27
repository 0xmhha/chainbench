package setup_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/setup"
)

func twoNodeSet() node.NodeSet {
	return node.NodeSet{
		Chain: "stablenet", Network: "local",
		Nodes: []node.Node{
			{Index: 1, Role: node.RoleValidator, PID: 1001},
			{Index: 5, Role: node.RoleEndpoint, PID: 1005},
		},
	}
}

func TestStopNode(t *testing.T) {
	fd := &fakeDriver{}
	if err := setup.StopNode(context.Background(), fd, twoNodeSet(), 5); err != nil {
		t.Fatalf("StopNode(5): %v", err)
	}
	if len(fd.stopped) != 1 || fd.stopped[0] != 5 {
		t.Errorf("stopped the wrong node: %v", fd.stopped)
	}

	// Unknown index and a node with no live PID both error, and neither stops
	// anything.
	fd2 := &fakeDriver{}
	if err := setup.StopNode(context.Background(), fd2, twoNodeSet(), 9); err == nil {
		t.Error("StopNode on an unknown index should error")
	}
	dead := node.NodeSet{Nodes: []node.Node{{Index: 1, PID: 0}}}
	if err := setup.StopNode(context.Background(), fd2, dead, 1); err == nil {
		t.Error("StopNode on a node with no live PID should error")
	}
	if len(fd2.stopped) != 0 {
		t.Errorf("errored StopNode still stopped something: %v", fd2.stopped)
	}
}

func TestRelaunchNode(t *testing.T) {
	fd := &fakeDriver{}
	spec := driver.NodeSpec{
		Index: 5, Role: node.RoleEndpoint, Host: "127.0.0.1",
		Ports: node.Endpoints{HTTP: 8505},
	}
	n, err := setup.RelaunchNode(context.Background(), fd, spec)
	if err != nil {
		t.Fatalf("RelaunchNode: %v", err)
	}
	// It re-provisions then launches the same index, and returns the refreshed
	// node with the driver's new PID and the spec's endpoint.
	if len(fd.provisioned) != 1 || fd.provisioned[0] != 5 || len(fd.launched) != 1 || fd.launched[0] != 5 {
		t.Errorf("relaunch drove wrong calls: provisioned=%v launched=%v", fd.provisioned, fd.launched)
	}
	if n.Index != 5 || n.PID != 1005 || n.RPCURL != "http://127.0.0.1:8505" {
		t.Errorf("refreshed node wrong: %+v", n)
	}
}
