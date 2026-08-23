package chainsetup

import (
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/wemix" // register the wemix plugin
	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// TestGovernanceSteps_MatchTheFamilysActions: the documented procedure and the
// procedure the code runs must be the same one. The runner executes these step
// ids as family actions by name, so a step renamed here without the family
// agreeing would report a step that never ran — and an action the family added
// without a step would run unreported.
func TestGovernanceSteps_MatchTheFamilysActions(t *testing.T) {
	p, err := registry.Get("wemix")
	if err != nil {
		t.Fatal(err)
	}
	roles := []node.Role{node.RoleBP, node.RoleBP}
	var declared []string
	for _, ph := range p.Family().BringUpPhases(roles) {
		declared = append(declared, ph.Actions...)
	}

	steps := map[string]bool{}
	for _, s := range governanceSteps() {
		steps[s.ID] = true
	}
	for _, a := range declared {
		if !steps[a] {
			t.Errorf("the family runs action %q, but the wemix case has no step for it — it would run unreported", a)
		}
	}
	for _, a := range []string{poa.ActionDeployGovernance, poa.ActionEtcdInit, poa.ActionVerifyEtcd, poa.ActionEtcdJoin} {
		if !steps[a] {
			t.Errorf("no step named %q", a)
		}
	}
}

// TestGovernanceSteps_PlaceBeforeGenesis pins the ordering the procedure
// actually requires. A governance member carries the address and port it will
// answer on, so a genesis generated before the placement exists would name
// places nothing was put — the tidier-looking order is the broken one.
func TestGovernanceSteps_PlaceBeforeGenesis(t *testing.T) {
	at := map[string]int{}
	for i, s := range governanceSteps() {
		at[s.ID] = i
	}
	for _, pair := range [][2]string{
		{"allocate", "wemix-genesis"},
		{"wemix-genesis", "provision"},
		{"launch-boot", poa.ActionDeployGovernance},
		{poa.ActionEtcdInit, poa.ActionVerifyEtcd},
		// The rest cannot join a cluster before they are running, and the boot
		// node cannot form one while they are.
		{poa.ActionVerifyEtcd, "launch-rest"},
		{"launch-rest", poa.ActionEtcdJoin},
	} {
		if at[pair[0]] >= at[pair[1]] {
			t.Errorf("%q must come before %q (positions %d and %d)", pair[0], pair[1], at[pair[0]], at[pair[1]])
		}
	}
}
