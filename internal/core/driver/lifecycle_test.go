package driver_test

import (
	"context"
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// recordingDriver records provisions/launches/stops without touching the OS.
// stopErr, when set, fails every Stop so the best-effort path is observable.
type recordingDriver struct {
	provisioned []int
	launched    []int
	stopped     []int
	stopErr     error
}

func (r *recordingDriver) Provision(_ context.Context, s driver.NodeSpec) error {
	r.provisioned = append(r.provisioned, s.Index)
	return nil
}

func (r *recordingDriver) Launch(_ context.Context, s driver.NodeSpec) (driver.Handle, error) {
	r.launched = append(r.launched, s.Index)
	return driver.Handle{Index: s.Index, PID: 1000 + s.Index}, nil
}

func (r *recordingDriver) Stop(_ context.Context, h driver.Handle) error {
	if r.stopErr != nil {
		return r.stopErr
	}
	r.stopped = append(r.stopped, h.Index)
	return nil
}

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
	d := &recordingDriver{}
	if err := driver.StopNode(context.Background(), d, twoNodeSet(), 5); err != nil {
		t.Fatalf("StopNode(5): %v", err)
	}
	if len(d.stopped) != 1 || d.stopped[0] != 5 {
		t.Errorf("stopped the wrong node: %v", d.stopped)
	}
}

func TestStopNode_UnknownIndexOrDeadPIDStopsNothing(t *testing.T) {
	d := &recordingDriver{}
	if err := driver.StopNode(context.Background(), d, twoNodeSet(), 9); err == nil {
		t.Error("StopNode on an unknown index should error")
	}
	dead := node.NodeSet{Nodes: []node.Node{{Index: 1, PID: 0}}}
	if err := driver.StopNode(context.Background(), d, dead, 1); err == nil {
		t.Error("StopNode on a node with no live PID should error")
	}
	if len(d.stopped) != 0 {
		t.Errorf("errored StopNode still stopped something: %v", d.stopped)
	}
}

func TestRelaunchNode(t *testing.T) {
	d := &recordingDriver{}
	spec := driver.NodeSpec{
		Index: 5, Role: node.RoleEndpoint, Host: "127.0.0.1",
		Ports: node.Endpoints{HTTP: 8505},
	}
	n, err := driver.RelaunchNode(context.Background(), d, spec)
	if err != nil {
		t.Fatalf("RelaunchNode: %v", err)
	}
	// It re-provisions then launches the same index, and returns the refreshed
	// node with the driver's new PID and the spec's endpoint.
	if len(d.provisioned) != 1 || d.provisioned[0] != 5 || len(d.launched) != 1 || d.launched[0] != 5 {
		t.Errorf("relaunch drove wrong calls: provisioned=%v launched=%v", d.provisioned, d.launched)
	}
	if n.Index != 5 || n.PID != 1005 || n.RPCURL != "http://127.0.0.1:8505" {
		t.Errorf("refreshed node wrong: %+v", n)
	}
}

func TestStopNodeSet_SkipsNodesWithNoPID(t *testing.T) {
	d := &recordingDriver{}
	ns := twoNodeSet()
	ns.Nodes = append(ns.Nodes, node.Node{Index: 7, PID: 0})

	stopped, errs := driver.StopNodeSet(context.Background(), d, ns)
	if stopped != 2 || len(errs) != 0 {
		t.Fatalf("stopped=%d errs=%v, want 2/none", stopped, errs)
	}
	if len(d.stopped) != 2 {
		t.Errorf("a node with no PID must not be stopped: %v", d.stopped)
	}
}

func TestStopNodeSet_IsBestEffort(t *testing.T) {
	// One failing Stop must not prevent the others from being attempted: a dead
	// PID is exactly when tearing the rest of the network down matters most.
	d := &recordingDriver{stopErr: errors.New("no such process")}

	stopped, errs := driver.StopNodeSet(context.Background(), d, twoNodeSet())
	if stopped != 0 {
		t.Errorf("stopped = %d, want 0", stopped)
	}
	if len(errs) != 2 {
		t.Fatalf("errs = %v, want one per node", errs)
	}
}
