package chainsetup

import (
	"slices"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/registry"
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
