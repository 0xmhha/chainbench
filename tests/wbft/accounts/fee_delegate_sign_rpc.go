// This file ports the fee-delegation signing RPC presence check from the legacy
// bash regression suite (regression/g-api/g5-01-sign-raw-fee-delegate).
//
// # Test: fee-delegate-sign-rpc-present
//
// Intent:   the chainbench-distinctive eth_signRawFeeDelegateTransaction method
//
//	is registered on the node's RPC surface (indirect check: the full
//	signing flow is exercised by fee-delegated-transfer).
//
// Applies:  stablenet, wbft. Requires: the "rpc" capability.
// Method:   call eth_signRawFeeDelegateTransaction with a throwaway argument. A
//
//	registered method rejects the bad input with a normal error (or returns
//	a result); only a method-not-found (-32601) response means it is absent.
//
// Pass:     the call does not return a method-not-found error.
//
// This is a read-only RPC probe: it makes no state change, so the sibling
// _test.go exercises it end to end against an httptest mock (returning a
// non-method-not-found error) in addition to the registration/gating checks.
package accounts

import (
	"encoding/json"
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

// feeDelegateSignFrom is an arbitrary address used only to shape a well-formed
// request; the method rejects it, which is fine — we only care that the method
// exists.
const feeDelegateSignFrom = "0x00000000000000000000000000000000C0FFEE08"

func init() {
	testkit.Register(testkit.Case{
		Name:         "fee-delegate-sign-rpc-present",
		Category:     "accounts",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           feeDelegateSignRPCPresent,
	})
}

func feeDelegateSignRPCPresent(t *testkit.T) {
	var out json.RawMessage
	err := t.Primary().Call(t.Ctx(), "eth_signRawFeeDelegateTransaction",
		&out, map[string]string{"from": feeDelegateSignFrom}, "0x00")
	t.Truef(!isMethodNotFoundErr(err),
		"eth_signRawFeeDelegateTransaction must be a registered method (got %v)", err)
}

// isMethodNotFoundErr reports whether err is a JSON-RPC "method not found"
// response (code -32601 or the geth phrasing), as opposed to any other error
// the method might legitimately return for a throwaway argument.
func isMethodNotFoundErr(err error) bool {
	if err == nil {
		return false
	}
	m := strings.ToLower(err.Error())
	return strings.Contains(m, "-32601") ||
		strings.Contains(m, "method not found") ||
		strings.Contains(m, "does not exist") ||
		strings.Contains(m, "not available")
}
