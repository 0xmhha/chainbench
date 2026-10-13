package testengine

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// TestAttachWithoutNodeControlOwnsNoProcess: a read-only attach adopts the
// workspace's network for reading and hands the steps no control over its
// processes, so a fault step fails instead of stopping a node it does not own.
func TestAttachWithoutNodeControlOwnsNoProcess(t *testing.T) {
	net := composed{control: workspaceNodes{dataDir: "/w"}}
	if got := (&runner{in: RunSuiteIn{}}).adopted(net); got.control == nil {
		t.Fatal("an ordinary attach lost its node control")
	}
	r := newRunner(chainsetup.Deps{}, attachRunSuiteIn(AttachWorkspaceIn{DataDir: "/w", NoNodeControl: true, ReadOnlyKeys: true}))
	if got := r.adopted(net); got.control != nil {
		t.Fatal("a read-only attach kept control over the node processes")
	}
	if !r.in.ReadOnlyKeys || !r.in.KeepUp {
		t.Fatalf("a read-only attach must keep the network up and its keys unchanged: %+v", r.in)
	}
}
