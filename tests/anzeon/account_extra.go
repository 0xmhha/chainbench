// This file ports the account-Extra bitmap cases (from regression
// h-hardfork h-30, h-33, h-34). A stablenet genesis alloc entry may carry an
// `extra` bitmap whose top bits seed AccountManager status at genesis-init:
// bit 62 = authorized, bit 63 = blacklisted. These cases run on a network
// launched with the account-extra overlay
// (pkg/chains/stablenet/overlays/account-extra.json, which advertises the
// "account-extra" capability), so the three fixture accounts exist.
//
// # Test: authorized-extra-bit-synced
//
// Intent:   an alloc account with Extra bit 62 is authorized in AccountManager.
// Applies:  stablenet. Requires "rpc" and "account-extra".
// Method:   AccountManager.isAuthorized(0x90F7…) == 1.
//
// # Test: blacklisted-extra-bit-synced
//
// Intent:   an alloc account with Extra bit 63 is blacklisted in AccountManager.
// Applies:  stablenet. Requires "rpc" and "account-extra".
// Method:   AccountManager.isBlacklisted(0x15d3…) == 1.
//
// # Test: dual-status-extra
//
// Intent:   an alloc account with Extra bits 62+63 is both authorized and
//
//	blacklisted, independently.
//
// Applies:  stablenet. Requires "rpc" and "account-extra".
// Method:   isAuthorized(0x9965…) == 1 AND isBlacklisted(0x9965…) == 1.
//
// # Test: extra-balance-preserved
//
// Intent:   syncing the Extra bits into the contracts must not alter balances.
// Applies:  stablenet. Requires "rpc" and "account-extra".
// Method:   balance(0x90F7…) == balance(0x15d3…) == 1e18 (the alloc balance).
//
// These are chainbench TEST CODE (requirement #16): they need a live network
// launched with the account-extra overlay, so the sibling _test.go validates
// registration and the capability gating (they skip without account-extra).
package anzeon

import (
	"math/big"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// account-extra fixture addresses (pkg/chains/stablenet/overlays/account-extra.json).
const (
	extraAuthorized  = "0x90F79bf6EB2c4f870365E785982E1f101E93b906" // bit 62
	extraBlacklisted = "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65" // bit 63
	extraDual        = "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc" // bits 62+63
	// extraAllocBalanceWei is each fixture account's alloc balance (0xDE0B6B3A7640000).
	extraAllocBalanceWei = "1000000000000000000"
)

func init() {
	reg := func(name string, fn func(*testkit.T)) {
		testkit.Register(testkit.Case{
			Name:         name,
			Category:     "system-contracts",
			ChainCompat:  []string{"stablenet"},
			RequiresCaps: []string{"rpc", "account-extra"},
			Fn:           fn,
		})
	}
	reg("authorized-extra-bit-synced", authorizedExtraBitSynced)
	reg("blacklisted-extra-bit-synced", blacklistedExtraBitSynced)
	reg("dual-status-extra", dualStatusExtra)
	reg("extra-balance-preserved", extraBalancePreserved)
}

// amStatus reads an AccountManager address-predicate (isAuthorized/isBlacklisted).
func amStatus(t *testkit.T, method, addr string) *big.Int {
	v, err := accounts.ReadUint(t.Ctx(), caller(t), accountManager, method, accounts.AddressArg(addr))
	t.NoErr(err, method)
	return v
}

func authorizedExtraBitSynced(t *testkit.T) {
	got := amStatus(t, "isAuthorized(address)", extraAuthorized)
	t.Truef(got.Cmp(big.NewInt(1)) == 0, "isAuthorized(%s) == 1 (Extra bit 62 synced), got %s", extraAuthorized, got)
}

func blacklistedExtraBitSynced(t *testkit.T) {
	got := amStatus(t, "isBlacklisted(address)", extraBlacklisted)
	t.Truef(got.Cmp(big.NewInt(1)) == 0, "isBlacklisted(%s) == 1 (Extra bit 63 synced), got %s", extraBlacklisted, got)
}

func dualStatusExtra(t *testkit.T) {
	auth := amStatus(t, "isAuthorized(address)", extraDual)
	t.Truef(auth.Cmp(big.NewInt(1)) == 0, "isAuthorized(%s) == 1 (Extra bit 62), got %s", extraDual, auth)
	bl := amStatus(t, "isBlacklisted(address)", extraDual)
	t.Truef(bl.Cmp(big.NewInt(1)) == 0, "isBlacklisted(%s) == 1 (Extra bit 63), got %s", extraDual, bl)
}

func extraBalancePreserved(t *testkit.T) {
	want, _ := new(big.Int).SetString(extraAllocBalanceWei, 10)
	for _, addr := range []string{extraAuthorized, extraBlacklisted} {
		var hexBal string
		t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBalance", &hexBal, addr, "latest"), "eth_getBalance")
		got := hexBig(t, hexBal, "balance")
		t.Truef(got.Cmp(want) == 0, "balance(%s) == %s (Extra sync preserved alloc balance), got %s", addr, want, got)
	}
}
