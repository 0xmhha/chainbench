// This file ports the authorized-account cases (from regression c-anzeon
// c-02 and e-blacklist-authorized e-09). An account authorized in the
// AccountManager is exempt from anzeon's tip forcing (it may set any priority
// fee), and every transaction it sends gets an AuthorizedTxExecuted event
// appended by the AccountManager.
//
// Unlike the bash originals, which authorize a preset node account, these cases
// authorize a FRESH generated account at runtime: fund it from the faucet, pass
// proposeAddAuthorizedAccount through GovCouncil to quorum, then act as that
// account. So no preset "authorized" key is needed — the bench grants the status
// via governance to an account it owns (same pattern as the blacklist cases).
//
// # Test: authorized-account-gastip-free
//
// Intent:   an authorized account may set its priority fee freely — anzeon does
//
//	NOT force it down to the header GasTip.
//
// Applies:  stablenet. Requires "rpc".
// Method:   authorize a fresh account; send a type-0x02 transfer with a tip 3x
//
//	the header GasTip; assert the tip charged equals the requested tip (not the
//	header GasTip) and an AuthorizedTxExecuted event is present.
//
// These are chainbench TEST CODE (requirement #16): live governance + tx flows,
// so the sibling _test.go validates registration/gating.
package anzeon

import (
	"math/big"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// authorizedTxExecutedTopic is the AccountManager AuthorizedTxExecuted event
// topic0 (pinned from the regression suite).
const authorizedTxExecutedTopic = "0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373"

func init() {
	testkit.Register(testkit.Case{
		Name:         "authorized-account-gastip-free",
		Category:     "gas-policy",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           authorizedAccountGastipFree,
	})
}

// authorizeFreshAccount generates a new account, funds it from the faucet, and
// authorizes it in the AccountManager via GovCouncil to quorum. It returns an
// opened wallet for the authorized account.
func authorizeFreshAccount(t *testkit.T) accounts.Wallet {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")

	key, addr, err := accounts.GenerateKey()
	t.NoErr(err, "generate key")

	// Fund the fresh account from the faucet so it can pay gas.
	fkey := fundedKey(t)
	fw, err := ap.OpenWallet(t.Ctx(), fkey, primary.RPCURL)
	t.NoErr(err, "open faucet wallet")
	fundHash, err := fw.SendCoin(t.Ctx(), addr, tenEther())
	t.NoErr(err, "fund fresh account")
	c := rpc.Dial(primary.RPCURL)
	t.WaitFor(func() bool { return receiptSucceeded(t.Ctx(), c, fundHash) },
		90*time.Second, time.Second, "fund receipt")

	// Grant the authorized status via governance, then wait until it takes.
	councilProposalToQuorum(t, "proposeAddAuthorizedAccount(address)", addr)
	t.WaitFor(func() bool {
		v, err := accounts.ReadUint(t.Ctx(), caller(t), accountManager,
			"isAuthorized(address)", accounts.AddressArg(addr))
		return err == nil && v.Cmp(big.NewInt(1)) == 0
	}, 90*time.Second, time.Second, "fresh account authorized")

	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open fresh wallet")
	return w
}

// tenEther is a generous gas float (10 ETH) for a fresh account.
func tenEther() *big.Int {
	e18 := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	return new(big.Int).Mul(big.NewInt(10), e18)
}

func authorizedAccountGastipFree(t *testkit.T) {
	w := authorizeFreshAccount(t)

	var latest string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_blockNumber", &latest), "eth_blockNumber")
	headerTip := headerGasTip(t, latest)
	t.Truef(headerTip.Sign() > 0, "header GasTip is positive (got %s)", headerTip)
	baseFee := latestBaseFee(t)

	// Request a tip 3x the header tip; an authorized account keeps it (not forced).
	customTip := new(big.Int).Mul(headerTip, big.NewInt(3))
	feeCap := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), customTip)

	hash, err := w.SendDynamicFeeGas(t.Ctx(), gastipRecipient, big.NewInt(1), feeCap, customTip)
	t.NoErr(err, "send authorized custom-tip tx")

	tipUsed := effectiveTip(t, hash)
	t.Truef(tipUsed.Cmp(customTip) == 0,
		"authorized account keeps its tip %s (charged %s; header GasTip was %s)",
		customTip, tipUsed, headerTip)

	primary, _ := t.NodeSet().Primary()
	c := rpc.Dial(primary.RPCURL)
	t.Truef(receiptHasTopic(t.Ctx(), c, hash, authorizedTxExecutedTopic),
		"AuthorizedTxExecuted event present for the authorized account's tx")
}
