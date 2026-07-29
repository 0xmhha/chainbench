// This file adds a system-contract bytecode read (ported from regression
// g-api g1-06), completing the readily-portable regression read surface.
//
// # Test: native-coin-adapter-code
//
// Intent:   the native-coin adapter (0x..1000) is a deployed system contract, so
//
//	eth_getCode at its address returns substantial non-empty bytecode.
//
// Applies:  stablenet (the go-stablenet system contracts). Requires "rpc".
// Method:   eth_getCode(nativeCoinAdapter, "latest"); assert the code is hex and
//
//	longer than a bare "0x".
//
// Pass:     the adapter has non-empty deployed code.
//
// This is chainbench TEST CODE (requirement #16): registered at init and run by
// the testrun phase (the sibling _test.go runs it against a mock node).
package anzeon

import (
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "native-coin-adapter-code",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           nativeCoinAdapterCode,
	})
}

func nativeCoinAdapterCode(t *testkit.T) {
	var code string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getCode", &code, nativeCoinAdapter, "latest"), "eth_getCode")
	t.Truef(strings.HasPrefix(code, "0x"), "code %q is 0x-hex", code)
	t.Truef(len(strings.TrimPrefix(code, "0x")) > 2, "adapter has non-empty deployed code (got %q)", code)
}
