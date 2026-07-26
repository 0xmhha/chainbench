// Package api holds JSON-RPC read-surface test cases (ported from the legacy
// bash regression suite tests/regression/g-api).
//
// # Test: block-by-hash-consistency
//
// Intent:   the node must return the same block whether looked up by number or
//
//	by hash — a basic RPC integrity check.
//
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   eth_getBlockByNumber("latest") -> take its number+hash, then
//
//	eth_getBlockByHash(hash) and assert the number and hash match.
//
// Pass:     the two lookups agree on number and hash.
//
// # Test: gas-price-positive
//
// Intent:   eth_gasPrice must return a sane, positive value.
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   eth_gasPrice; parse the hex; assert > 0.
// Pass:     gas price parses and is greater than zero.
//
// These are chainbench TEST CODE (requirement #16): registered at init and run
// by the testrun phase against a live NodeSet (the sibling _test.go validates
// registration and runs each against a mock node).
package api

import (
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

type blockRef struct {
	Number string `json:"number"`
	Hash   string `json:"hash"`
}

func init() {
	testkit.Register(testkit.Case{
		Name:         "block-by-hash-consistency",
		Category:     "api",
		RequiresCaps: []string{"rpc"},
		Fn:           blockByHashConsistency,
	})
	testkit.Register(testkit.Case{
		Name:         "gas-price-positive",
		Category:     "api",
		RequiresCaps: []string{"rpc"},
		Fn:           gasPricePositive,
	})
}

func blockByHashConsistency(t *testkit.T) {
	cli := t.Primary()
	var byNum blockRef
	t.NoErr(cli.Call(t.Ctx(), "eth_getBlockByNumber", &byNum, "latest", false), "eth_getBlockByNumber(latest)")
	t.Truef(byNum.Hash != "" && byNum.Number != "", "latest block has number and hash")

	var byHash blockRef
	t.NoErr(cli.Call(t.Ctx(), "eth_getBlockByHash", &byHash, byNum.Hash, false), "eth_getBlockByHash")
	t.Equalf(byHash.Number, byNum.Number, "block number matches across number/hash lookup")
	t.Equalf(byHash.Hash, byNum.Hash, "block hash matches across number/hash lookup")
}

func gasPricePositive(t *testkit.T) {
	var hexPrice string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_gasPrice", &hexPrice), "eth_gasPrice")
	price, ok := new(big.Int).SetString(strings.TrimPrefix(hexPrice, "0x"), 16)
	t.Truef(ok, "gas price %q parses as hex", hexPrice)
	t.Truef(price != nil && price.Sign() > 0, "gas price is positive (got %s)", hexPrice)
}
