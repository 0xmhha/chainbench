package chainsetup

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// TestCountValidators_CanonicalSpelling: this number sizes the genesis
// validator set. Reading zero from a canonical plan would build a chain with no
// producers.
func TestCountValidators_CanonicalSpelling(t *testing.T) {
	reqs := []node.LaunchReq{
		{Role: node.RoleBP},
		{Role: node.RoleValidator},
		{Role: node.RoleEN},
		{Role: node.RolePN},
	}
	if got := countValidators(reqs); got != 2 {
		t.Fatalf("countValidators = %d, want 2 (both spellings of the producing role)", got)
	}
	specs := driver.Plan{Nodes: []driver.NodeSpec{
		{Index: 1, Role: node.RoleBP}, {Index: 2, Role: node.RoleEndpoint}, {Index: 3, Role: node.RoleValidator},
	}}
	if got := planValidatorCount(specs); got != 2 {
		t.Fatalf("planValidatorCount = %d, want 2", got)
	}
}
