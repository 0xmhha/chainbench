package node_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestLayout_CompositionIsolation is W3's datadir contract: a composition id
// isolates node datadirs under NodesDir/<id>, while no id keeps the flat layout
// every earlier workspace used.
func TestLayout_CompositionIsolation(t *testing.T) {
	flat := node.Layout{Root: "/data"}
	if got := flat.DataDir("node1"); got != "/data/node1" {
		t.Fatalf("no id must stay flat: %q", got)
	}
	iso := node.Layout{Root: "/data", CompositionID: "abc123def456", NodesDir: "node"}
	if got := iso.DataDir("node1"); got != "/data/node/abc123def456/node1" {
		t.Fatalf("isolated datadir = %q, want under node/<id>", got)
	}
	// The nodekey and keystore follow the datadir, so isolation carries through.
	if got := iso.NodekeyPath("node1"); got != "/data/node/abc123def456/node1/nodekey" {
		t.Fatalf("nodekey not under the isolated datadir: %q", got)
	}
	// An empty NodesDir defaults to "node".
	d := node.Layout{Root: "/r", CompositionID: "x"}
	if got := d.DataDir("bp1"); got != "/r/node/x/bp1" {
		t.Fatalf("default nodes dir = %q", got)
	}
}
