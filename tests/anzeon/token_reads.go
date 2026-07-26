// This file adds address-argument system-contract read cases (ported from
// tests/regression f-system-contracts f1-02 balance-of and c-anzeon
// authorization), exercising the ABI address-arg encoding in pkg/accounts.
//
// # Test: token-balance-readable
//
// Intent:   the native-coin adapter answers balanceOf(address) and the value is
//
//	consistent with totalSupply (a balance cannot exceed the supply).
//
// Applies:  stablenet. Requires: "rpc".
// Method:   eth_call balanceOf(addr) and totalSupply(); assert both decode and
//
//	totalSupply >= balance.
// Pass:     both decode and the invariant holds.
//
// # Test: account-authorization-readable
//
// Intent:   the account manager answers isAuthorized(address) with a boolean
//
//	word (0 or 1) — the read behind anzeon's gas-tip authorization.
//
// Applies:  stablenet. Requires: "rpc".
// Method:   eth_call isAuthorized(addr); assert the word is 0 or 1.
// Pass:     the returned word is 0 or 1.

package anzeon

import (
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

const accountManager = "0x0000000000000000000000000000000000B00003"

// sampleAccount is a genesis validator address used as a read argument.
const sampleAccount = "0xc17d493883eaa3b4cceb0f214b273392d562f9d8"

func init() {
	testkit.Register(testkit.Case{
		Name:         "token-balance-readable",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           tokenBalanceReadable,
	})
	testkit.Register(testkit.Case{
		Name:         "account-authorization-readable",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           accountAuthorizationReadable,
	})
}

func tokenBalanceReadable(t *testkit.T) {
	bal := ethCallWord(t, nativeCoinAdapter, accounts.EncodeCall("balanceOf(address)", accounts.AddressArg(sampleAccount)))
	sup := ethCallWord(t, nativeCoinAdapter, accounts.EncodeCall("totalSupply()"))
	t.Truef(sup.Cmp(bal) >= 0, "totalSupply (%s) >= balance (%s)", sup, bal)
}

func accountAuthorizationReadable(t *testkit.T) {
	v := ethCallWord(t, accountManager, accounts.EncodeCall("isAuthorized(address)", accounts.AddressArg(sampleAccount)))
	t.Truef(v.Sign() == 0 || v.Cmp(big.NewInt(1)) == 0, "isAuthorized returns 0 or 1 (got %s)", v)
}

// ethCallWord performs an eth_call and decodes the returned 32-byte word as a
// uint256, failing the case if the call errors or the word is malformed.
func ethCallWord(t *testkit.T, to, data string) *big.Int {
	var ret string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_call", &ret,
		map[string]any{"to": to, "data": data}, "latest"), "eth_call "+data[:10])
	v, ok := new(big.Int).SetString(strings.TrimPrefix(ret, "0x"), 16)
	t.Truef(ok, "eth_call result %q decodes as a uint256", ret)
	if v == nil {
		return big.NewInt(0)
	}
	return v
}
