// # Test: validator-set-count
//
// Intent:   the engine must report at least as many validators as the NodeSet
//
//	runs validator-role nodes — a smaller set means a validator dropped
//	out of consensus (ported from regression/g-api/
//	g3-02-get-validators.sh).
//
// Applies:  stablenet, wbft. Requires: "rpc", "consensus".
// Method:   istanbul_getValidators("latest"); count the returned addresses and
//
//	compare against the number of validator-role nodes in the set.
//
// Pass:     the validator set is non-empty and its size >= the count of
//
//	validator-role nodes.
//
// These are chainbench TEST CODE (requirement #16): registered at init and run
// by the testrun phase against a live NodeSet (the sibling _test.go validates
// registration and runs each against a mock node).
package consensus

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "validator-set-count",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc", "consensus"},
		Fn:           validatorSetCount,
	})
}

func validatorSetCount(t *testkit.T) {
	want := 0
	for _, n := range t.NodeSet().Nodes {
		if n.Role == node.RoleValidator {
			want++
		}
	}
	var vals []string
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getValidators", &vals, "latest"), "istanbul_getValidators")
	t.Truef(len(vals) > 0, "validator set is non-empty (got %d)", len(vals))
	t.Truef(len(vals) >= want,
		"validator set (%d) covers all validator-role nodes (%d)", len(vals), want)
}
