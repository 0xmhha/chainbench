// Package anzeon holds go-stablenet system-contract test cases (ported from the
// legacy regression tests/regression/f-system-contracts and c-anzeon), exercised
// through eth_call with the ABI helpers in pkg/accounts.
//
// # Test: system-contracts-deployed
//
// Intent:   the anzeon system contracts (governance + native-coin managers) must
//
//	be deployed at their fixed addresses — the chain cannot enforce
//	governance/fees otherwise.
//
// Applies:  stablenet. Requires: "rpc".
// Method:   eth_getCode at each fixed system-contract address; assert non-empty.
// Pass:     every listed address has deployed code.
//
// # Test: token-total-supply-readable
//
// Intent:   the native-coin adapter (WKRC) answers the ERC-20 totalSupply() read.
// Applies:  stablenet. Requires: "rpc".
// Method:   eth_call NATIVE_COIN_ADAPTER.totalSupply(); assert a 32-byte word is
//
//	returned and decodes as a uint256.
//
// Pass:     totalSupply returns a decodable word.
//
// These are chainbench TEST CODE (requirement #16): run by the testrun phase
// against a live NodeSet (the sibling _test.go runs them against a mock).
package anzeon

import (
	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// Fixed anzeon system-contract addresses (tests/regression/lib/common.sh).
var systemContracts = map[string]string{
	"native-coin-adapter": "0x0000000000000000000000000000000000001000",
	"gov-validator":       "0x0000000000000000000000000000000000001001",
	"gov-master-minter":   "0x0000000000000000000000000000000000001002",
	"gov-minter":          "0x0000000000000000000000000000000000001003",
	"gov-council":         "0x0000000000000000000000000000000000001004",
	"account-manager":     "0x0000000000000000000000000000000000B00003",
	"native-coin-manager": "0x0000000000000000000000000000000000B00002",
	"bls-pop":             "0x0000000000000000000000000000000000B00001",
}

const nativeCoinAdapter = "0x0000000000000000000000000000000000001000"

func init() {
	testkit.Register(testkit.Case{
		Name:         "system-contracts-deployed",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           systemContractsDeployed,
	})
	testkit.Register(testkit.Case{
		Name:         "token-total-supply-readable",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           tokenTotalSupplyReadable,
	})
}

func systemContractsDeployed(t *testkit.T) {
	for name, addr := range systemContracts {
		var code string
		t.NoErr(t.Primary().Call(t.Ctx(), "eth_getCode", &code, addr, "latest"), "eth_getCode "+name)
		t.Truef(code != "" && code != "0x", "%s (%s) has deployed code", name, addr)
	}
}

func tokenTotalSupplyReadable(t *testkit.T) {
	sup, err := accounts.ReadUint(t.Ctx(), caller(t), nativeCoinAdapter, "totalSupply()")
	t.NoErr(err, "totalSupply")
	t.Truef(sup.Sign() >= 0, "totalSupply decodes as a non-negative uint256 (got %s)", sup)
}
