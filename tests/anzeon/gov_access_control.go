// This file ports the governance access-control rejection cases (from
// regression f-system-contracts f5-05, f4-04). A non-member account that
// calls a member-only system-contract method directly must be rejected — either
// at submission or by an on-chain revert (status 0x0). The bench uses a FRESH
// funded account (never a member) as the caller.
//
// # Test: direct-blacklist-call-rejected (f5-05)
//
// Intent:   a non-member calling AccountManager.blacklist directly is rejected.
// Applies:  stablenet. Requires "rpc".
// Method:   fund a fresh account; call blacklist(address); expect submission
//
//	rejection or a reverted receipt (never a successful mine).
//
// # Test: non-member-configure-minter-rejected (f4-04)
//
// Intent:   a non-member calling GovMasterMinter.proposeConfigureMinter is
//
//	rejected by the onlyActiveMember guard.
//
// Applies:  stablenet. Requires "rpc".
// Method:   fund a fresh account; call proposeConfigureMinter(address,uint256);
//
//	expect submission rejection or a reverted receipt.
//
// These are chainbench TEST CODE (requirement #16): live transactions, so the
// sibling _test.go validates registration/gating.
package anzeon

import (
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func init() {
	reg := func(name string, fn func(*testkit.T)) {
		testkit.Register(testkit.Case{
			Name:         name,
			Category:     "system-contracts",
			ChainCompat:  []string{"stablenet"},
			RequiresCaps: []string{"rpc"},
			Fn:           fn,
		})
	}
	reg("direct-blacklist-call-rejected", directBlacklistCallRejected)
	reg("non-member-configure-minter-rejected", nonMemberConfigureMinterRejected)
}

// newFundedWallet generates a fresh account, funds it from the faucet, and opens
// its wallet. The account is not a governance member, so it is the right caller
// for access-control rejection cases.
func newFundedWallet(t *testkit.T) accounts.Wallet {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")

	key, addr, err := accounts.GenerateKey()
	t.NoErr(err, "generate key")
	fkey := fundedKey(t)
	fw, err := ap.OpenWallet(t.Ctx(), fkey, primary.RPCURL)
	t.NoErr(err, "open faucet wallet")
	fundHash, err := fw.SendCoin(t.Ctx(), addr, tenEther())
	t.NoErr(err, "fund fresh account")
	c := rpc.Dial(primary.RPCURL)
	t.WaitFor(func() bool { return receiptSucceeded(t.Ctx(), c, fundHash) },
		90*time.Second, time.Second, "fund receipt")

	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open fresh wallet")
	return w
}

// mustRejectOrRevert asserts a member-only call from a non-member either fails at
// submission or mines as a revert — it must NOT succeed. dataHex is the 0x-hex
// calldata from EncodeCallArgs.
func mustRejectOrRevert(t *testkit.T, w accounts.Wallet, to, dataHex, what string) {
	data, err := hex.DecodeString(strings.TrimPrefix(dataHex, "0x"))
	t.NoErr(err, "decode calldata")
	hash, err := w.Execute(t.Ctx(), to, data, nil)
	if err != nil {
		return // rejected at submission — access control enforced
	}
	primary, _ := t.NodeSet().Primary()
	c := rpc.Dial(primary.RPCURL)
	// receiptReverted only returns true for status 0x0; a successful mine (0x1)
	// leaves this waiting until timeout, failing the case as intended.
	t.WaitFor(func() bool { return receiptReverted(t.Ctx(), c, hash) },
		60*time.Second, time.Second, what)
}

func directBlacklistCallRejected(t *testkit.T) {
	w := newFundedWallet(t)
	_, target, err := accounts.GenerateKey()
	t.NoErr(err, "generate target")
	data := accounts.EncodeCallArgs("blacklist(address)", accounts.Address(target))
	mustRejectOrRevert(t, w, accountManager, data, "direct AccountManager.blacklist from a non-member reverts")
}

func nonMemberConfigureMinterRejected(t *testkit.T) {
	w := newFundedWallet(t)
	_, minter, err := accounts.GenerateKey()
	t.NoErr(err, "generate minter")
	allowance := new(big.Int).Mul(big.NewInt(10), big.NewInt(1_000_000_000_000_000_000))
	data := accounts.EncodeCallArgs("proposeConfigureMinter(address,uint256)",
		accounts.Address(minter), accounts.Uint(allowance))
	mustRejectOrRevert(t, w, govMasterMinter, data, "non-member proposeConfigureMinter reverts")
}
