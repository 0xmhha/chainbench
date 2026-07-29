// This file adds JSON-RPC inspection read cases ported from the legacy bash
// regression suite (regression/g-api g1-01, g2-03, g4-03).
//
// # Test: block-transactions-field
//
// Intent:   eth_getBlockByNumber("latest", false) returns a well-formed block
//
//	header — hex number and hash — and always carries a transactions array
//	field (the read behind block explorers and sync checks).
//
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   eth_getBlockByNumber("latest", false); assert number and hash parse
//
//	as hex and the "transactions" field is present.
//
// Pass:     number+hash are hex and the transactions field exists.
//
// # Test: fee-history-well-formed
//
// Intent:   eth_feeHistory answers with a non-empty baseFeePerGas series, the
//
//	read behind EIP-1559 fee estimation.
//
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   eth_feeHistory("0x10", "latest", []); assert baseFeePerGas is
//
//	non-empty and every entry parses as a hex number.
//
// Pass:     baseFeePerGas is non-empty and every entry parses as hex.
//
// # Test: txpool-content-well-formed
//
// Intent:   txpool_content decodes and carries both a pending and a queued map —
//
//	the read behind transaction-pool inspection.
//
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   txpool_content; assert the result has both "pending" and "queued".
// Pass:     both the pending and queued maps are present.
//
// These are chainbench TEST CODE (requirement #16): registered at init and run
// by the testrun phase (the sibling _test.go runs each against a mock node).
package api

import (
	"encoding/json"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "block-transactions-field",
		Category:     "api",
		RequiresCaps: []string{"rpc"},
		Fn:           blockTransactionsField,
	})
	testkit.Register(testkit.Case{
		Name:         "fee-history-well-formed",
		Category:     "api",
		RequiresCaps: []string{"rpc"},
		Fn:           feeHistoryWellFormed,
	})
	testkit.Register(testkit.Case{
		Name:         "txpool-content-well-formed",
		Category:     "api",
		RequiresCaps: []string{"rpc"},
		Fn:           txpoolContentWellFormed,
	})
}

func blockTransactionsField(t *testkit.T) {
	var block map[string]json.RawMessage
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &block, "latest", false),
		"eth_getBlockByNumber(latest,false)")

	var number, hash string
	t.NoErr(json.Unmarshal(block["number"], &number), "decode block.number")
	t.NoErr(json.Unmarshal(block["hash"], &hash), "decode block.hash")
	t.Truef(isHexNumber(number), "block number %q is a hex number", number)
	t.Truef(isHexNumber(hash), "block hash %q is hex", hash)

	_, hasTx := block["transactions"]
	t.Truef(hasTx, "block has a transactions field")
}

func feeHistoryWellFormed(t *testkit.T) {
	var hist struct {
		BaseFeePerGas []string `json:"baseFeePerGas"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_feeHistory", &hist, "0x10", "latest", []any{}),
		"eth_feeHistory")
	t.Truef(len(hist.BaseFeePerGas) > 0, "baseFeePerGas is non-empty")
	for i, b := range hist.BaseFeePerGas {
		t.Truef(isHexNumber(b), "baseFeePerGas[%d]=%q is a hex number", i, b)
	}
}

func txpoolContentWellFormed(t *testkit.T) {
	var content map[string]json.RawMessage
	t.NoErr(t.Primary().Call(t.Ctx(), "txpool_content", &content), "txpool_content")
	_, hasPending := content["pending"]
	_, hasQueued := content["queued"]
	t.Truef(hasPending, "txpool_content has a pending map")
	t.Truef(hasQueued, "txpool_content has a queued map")
}
