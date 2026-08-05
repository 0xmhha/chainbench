// This file adds two JSON-RPC read cases ported from the legacy bash regression
// suite (regression a-ethereum a3-04, a4-04).
//
// # Test: estimate-gas
//
// Intent:   eth_estimateGas returns a sane estimate — at least the 21000
//
//	intrinsic gas of a plain value transfer.
//
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   eth_estimateGas({to, value:0x0}); parse the hex; assert >= 21000.
// Pass:     the estimate parses and is at least 21000.
//
// # Test: logs-query-well-formed
//
// Intent:   eth_getLogs answers a range filter with a well-formed log array (each
//
//	entry carries an address and topics), the read behind event queries.
//
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   eth_getLogs({fromBlock:latest, toBlock:latest}); assert each entry
//
//	has a 20-byte address.
//
// Pass:     the response decodes and every entry has a valid address.
//
// These are chainbench TEST CODE (requirement #16): registered at init and run
// by the testrun phase (the sibling _test.go runs each against a mock node).
package api

import (
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/internal/testkit"
)

// estimateTo is an arbitrary recipient for the gas estimate.
const estimateTo = "0x00000000000000000000000000000000C0FFEE0A"

func init() {
	testkit.Register(testkit.Case{
		Name:         "estimate-gas",
		Category:     "api",
		RequiresCaps: []string{"rpc"},
		Fn:           estimateGas,
	})
	testkit.Register(testkit.Case{
		Name:         "logs-query-well-formed",
		Category:     "api",
		RequiresCaps: []string{"rpc"},
		Fn:           logsQueryWellFormed,
	})
}

func estimateGas(t *testkit.T) {
	var hexGas string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_estimateGas",
		&hexGas, map[string]string{"to": estimateTo, "value": "0x0"}), "eth_estimateGas")
	gas, ok := new(big.Int).SetString(strings.TrimPrefix(hexGas, "0x"), 16)
	t.Truef(ok, "gas estimate %q parses as hex", hexGas)
	t.Truef(gas.Cmp(big.NewInt(21000)) >= 0, "gas estimate >= 21000 (got %s)", gas)
}

func logsQueryWellFormed(t *testkit.T) {
	var logs []struct {
		Address string   `json:"address"`
		Topics  []string `json:"topics"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getLogs",
		&logs, map[string]string{"fromBlock": "latest", "toBlock": "latest"}), "eth_getLogs")
	for i, l := range logs {
		t.Truef(strings.HasPrefix(l.Address, "0x") && len(l.Address) == 42,
			"log %d has a valid 20-byte address (got %q)", i, l.Address)
	}
}
