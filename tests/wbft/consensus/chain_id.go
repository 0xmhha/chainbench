// Package consensus holds wbft-family consensus test cases.
//
// # Test: chain-id
//
// Intent:   verify the network reports a non-zero, stable chain id over RPC —
//
//	the most basic liveness/identity check before deeper consensus
//	assertions.
//
// Applies:  stablenet, wbft (the wbft consensus family).
// Requires: the "rpc" capability.
// Method:   query eth_chainId on the primary node; assert it is non-zero and
//
//	equal across a second read (stable identity).
//
// Pass:     chain id > 0 and identical on both reads.
//
// This is chainbench TEST CODE (requirement #16): it lives under tests/, is
// named tests/<family>/<category>/<name>.go, carries this godoc header, and
// registers a testkit.Case at init — it is executed by the testrun phase
// against a live NodeSet, not by `go test` (the sibling _test.go validates the
// registration/convention).
package consensus

import (
	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "chain-id",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           chainID,
	})
}

func chainID(t *testkit.T) {
	c := t.Primary()
	first, err := c.ChainID(t.Ctx())
	t.NoErr(err, "eth_chainId (first read)")
	t.Truef(first > 0, "chain id must be non-zero, got %d", first)

	second, err := c.ChainID(t.Ctx())
	t.NoErr(err, "eth_chainId (second read)")
	t.Equalf(second, first, "chain id must be stable across reads")
}
