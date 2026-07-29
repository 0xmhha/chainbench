// # Test: validator-set-nonempty
//
// Intent:   the wbft-family consensus RPC must report a non-empty validator set
//
//	— an empty set means the engine cannot make progress (ported from the
//	legacy regression regression/b-wbft/b-07-istanbul-get-validators).
//
// Applies:  stablenet, wbft (the istanbul namespace). Requires: "rpc".
// Method:   istanbul_getValidators("latest"); assert the returned list is
//
//	non-empty and every entry is a 0x-prefixed address.
// Pass:     at least one validator is returned and all look like addresses.

package consensus

import (
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "validator-set-nonempty",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           validatorSetNonEmpty,
	})
}

func validatorSetNonEmpty(t *testkit.T) {
	var vals []string
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getValidators", &vals, "latest"), "istanbul_getValidators")
	t.Truef(len(vals) > 0, "validator set is non-empty (got %d)", len(vals))
	for _, v := range vals {
		t.Truef(strings.HasPrefix(v, "0x") && len(v) == 42, "validator %q looks like an address", v)
	}
}
