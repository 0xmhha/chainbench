package process_test

import (
	"context"
	"errors"
	"github.com/0xmhha/chainbench/internal/core/process"
	"testing"

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

func (r *recordingDriver) Provision(_ context.Context, s process.NodeSpec) error {
	r.provisioned = append(r.provisioned, s.Index)
	return nil
}

func (r *recordingDriver) Launch(_ context.Context, s process.NodeSpec) (process.Handle, error) {
	r.launched = append(r.launched, s.Index)
	return process.Handle{Index: s.Index, PID: 1000 + s.Index}, nil
}

func (r *recordingDriver) Stop(_ context.Context, h process.Handle) error {
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

func TestStopNodeSet_SkipsNodesWithNoPID(t *testing.T) {
	d := &recordingDriver{}
	ns := twoNodeSet()
	ns.Nodes = append(ns.Nodes, node.Node{Index: 7, PID: 0})

	stopped, errs := process.StopNodeSet(context.Background(), d, ns)
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

	stopped, errs := process.StopNodeSet(context.Background(), d, twoNodeSet())
	if stopped != 0 {
		t.Errorf("stopped = %d, want 0", stopped)
	}
	if len(errs) != 2 {
		t.Fatalf("errs = %v, want one per node", errs)
	}
}
