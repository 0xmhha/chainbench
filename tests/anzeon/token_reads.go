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

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
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
	call := caller(t)
	bal, err := accounts.ReadUint(t.Ctx(), call, nativeCoinAdapter, "balanceOf(address)", accounts.AddressArg(sampleAccount))
	t.NoErr(err, "balanceOf")
	sup, err := accounts.ReadUint(t.Ctx(), call, nativeCoinAdapter, "totalSupply()")
	t.NoErr(err, "totalSupply")
	t.Truef(sup.Cmp(bal) >= 0, "totalSupply (%s) >= balance (%s)", sup, bal)
}

func accountAuthorizationReadable(t *testkit.T) {
	v, err := accounts.ReadUint(t.Ctx(), caller(t), accountManager, "isAuthorized(address)", accounts.AddressArg(sampleAccount))
	t.NoErr(err, "isAuthorized")
	t.Truef(v.Sign() == 0 || v.Cmp(big.NewInt(1)) == 0, "isAuthorized returns 0 or 1 (got %s)", v)
}

// caller returns an accounts.EthCaller bound to the primary node.
func caller(t *testkit.T) accounts.EthCaller {
	p, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	return rpc.Dial(p.RPCURL).EthCall
}
