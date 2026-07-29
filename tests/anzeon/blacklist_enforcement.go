// This file ports the blacklist-enforcement cases (from regression
// e-blacklist-authorized e-01, e-02): a blacklisted sender or recipient must have
// its transfer rejected. Each case blacklists a FRESH generated address via
// governance (non-destructive — no shared fixture account is affected), so it is
// self-contained.
//
// # Test: sender-blacklisted-rejected
//
// Intent:   a transfer from a blacklisted account is rejected.
//
// Applies:  stablenet. Requires "rpc".
// Method:   generate a key; blacklist its address via GovCouncil to quorum; open
//
//	its wallet and SendCoin — the accounts guard rejects a blacklisted sender.
//
// Pass:     SendCoin returns a "blacklisted" error.
//
// # Test: recipient-blacklisted-rejected
//
// Intent:   a transfer to a blacklisted account is rejected.
//
// Applies:  stablenet. Requires "rpc".
// Method:   blacklist a fresh recipient address; SendCoin from the faucet wallet
//
//	to it — the accounts guard rejects a blacklisted recipient.
//
// Pass:     SendCoin returns a "blacklisted" error.
//
// These are chainbench TEST CODE (requirement #16): live multi-validator flows
// (the sibling _test.go validates registration/gating).
package anzeon

import (
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "sender-blacklisted-rejected",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           senderBlacklistedRejected,
	})
	testkit.Register(testkit.Case{
		Name:         "recipient-blacklisted-rejected",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           recipientBlacklistedRejected,
	})
	testkit.Register(testkit.Case{
		Name:         "unblacklist-restores",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           unblacklistRestores,
	})
}

// unblacklistRestores blacklists a fresh address, then removes the blacklist via
// governance and confirms the account manager no longer reports it blacklisted
// (regression e-04).
func unblacklistRestores(t *testkit.T) {
	_, target, err := accounts.GenerateKey()
	t.NoErr(err, "generate target address")
	blacklistAndWait(t, target)

	proposer := councilProposalToQuorum(t, "proposeRemoveBlacklist(address)", target)
	t.WaitFor(func() bool {
		v, err := accounts.ReadUint(t.Ctx(), proposer.client.EthCall, accountManager,
			"isBlacklisted(address)", accounts.AddressArg(target))
		return err == nil && v.Sign() == 0
	}, 60*time.Second, time.Second, "target is no longer blacklisted")
}

// blacklistAndWait blacklists target via GovCouncil and waits until the account
// manager reports it blacklisted.
func blacklistAndWait(t *testkit.T, target string) {
	proposer := councilProposalToQuorum(t, "proposeAddBlacklist(address)", target)
	t.WaitFor(func() bool {
		v, err := accounts.ReadUint(t.Ctx(), proposer.client.EthCall, accountManager,
			"isBlacklisted(address)", accounts.AddressArg(target))
		return err == nil && v.Cmp(big.NewInt(1)) == 0
	}, 60*time.Second, time.Second, "target is blacklisted")
}

func senderBlacklistedRejected(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	key, addr, err := accounts.GenerateKey()
	t.NoErr(err, "generate key")
	blacklistAndWait(t, addr)

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	_, err = w.SendCoin(t.Ctx(), "0x00000000000000000000000000000000C0FFEE10", big.NewInt(1))
	t.Truef(err != nil && strings.Contains(strings.ToLower(err.Error()), "blacklist"),
		"a blacklisted sender's transfer must be rejected (got %v)", err)
}

func recipientBlacklistedRejected(t *testkit.T) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	_, recipient, err := accounts.GenerateKey()
	t.NoErr(err, "generate recipient address")
	blacklistAndWait(t, recipient)

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")
	key, err := hex.DecodeString(faucetKeyHex)
	t.NoErr(err, "decode faucet key")
	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")

	_, err = w.SendCoin(t.Ctx(), recipient, big.NewInt(1))
	t.Truef(err != nil && strings.Contains(strings.ToLower(err.Error()), "blacklist"),
		"a transfer to a blacklisted recipient must be rejected (got %v)", err)
}
