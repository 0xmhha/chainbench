package chainsetup

import (
	"slices"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Where a network runs is a fact about the target, not about the chain, so no
// manifest can declare it. The composed network advertises it instead, and a
// case that needs a shell on a node's machine — one that programs a firewall
// to split the network in two — asks for it and is skipped elsewhere.
func TestNetworkCapabilities_SaysWhereTheNetworkRuns(t *testing.T) {
	m := registry.Manifest{Capabilities: []string{"rpc"}}

	remote := networkCapabilities(m, nil, GenesisOpts{}, true)
	if !slices.Contains(remote, registry.TargetRemote) {
		t.Errorf("remote network advertises %v, want %s among them", remote, registry.TargetRemote)
	}
	if slices.Contains(remote, registry.TargetLocal) {
		t.Errorf("remote network also claims %s: %v", registry.TargetLocal, remote)
	}

	local := networkCapabilities(m, nil, GenesisOpts{}, false)
	if !slices.Contains(local, registry.TargetLocal) {
		t.Errorf("local network advertises %v, want %s among them", local, registry.TargetLocal)
	}
	if slices.Contains(local, registry.TargetRemote) {
		t.Errorf("local network also claims %s: %v", registry.TargetRemote, local)
	}
}

// A requirement is refused when its prefix is unknown, which is how a typo is
// told from an unmet requirement. target: has to be known, or a case asking for
// it would be reported as misspelled rather than gated.
func TestTargetIsAKnownRequirementPrefix(t *testing.T) {
	for _, req := range []string{registry.TargetRemote, registry.TargetLocal} {
		if why := registry.MalformedCapability(req); why != "" {
			t.Errorf("%s is refused as a requirement: %s", req, why)
		}
	}
	if registry.MalformedCapability("targt:remote") == "" {
		t.Error("a misspelled prefix was accepted")
	}
}

// TestEveryNodeIsRemote_AMixedSetIsNotRemote pins the fix for a set whose
// entries are not all the same kind.
//
// The workspace Target comes from the set's FIRST entry under --all-servers, so
// before this the answer followed file order: one remote entry above fourteen
// loopback ones advertised target:remote, and reordering the same file flipped
// it. A case that needs a separate machine per group would have been let
// through to fail where it could not split the network at all.
func TestEveryNodeIsRemote_AMixedSetIsNotRemote(t *testing.T) {
	cases := []struct {
		name    string
		servers []string // per node: the server-set entry, "" for this machine
		want    bool
	}{
		{"every node on a server", []string{"s1", "s2", "s3"}, true},
		{"one node here", []string{"s1", "", "s3"}, false},
		{"the local one first", []string{"", "s2", "s3"}, false},
		{"the local one last", []string{"s1", "s2", ""}, false},
		{"no node on a server", []string{"", "", ""}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := &Workspace{}
			for i, srv := range tc.servers {
				w.state.Nodes = append(w.state.Nodes, node.Record{Index: i + 1, Server: srv})
			}
			if got := w.everyNodeIsRemote(); got != tc.want {
				t.Errorf("everyNodeIsRemote() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestEveryNodeIsRemote_NoPlacementFallsBackToTarget keeps the shapes that have
// no node table — before place, and the single-server and no-server-set runs —
// answering as they always did.
func TestEveryNodeIsRemote_NoPlacementFallsBackToTarget(t *testing.T) {
	local := &Workspace{}
	if local.everyNodeIsRemote() {
		t.Error("an empty placement with a local target is not remote")
	}
	remote := &Workspace{}
	remote.state.Target = resource.Spec{Server: "s1"}
	if !remote.everyNodeIsRemote() {
		t.Error("an empty placement with a server target is remote")
	}
}
