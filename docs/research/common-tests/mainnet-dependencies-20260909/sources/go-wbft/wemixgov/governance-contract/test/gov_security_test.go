// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright 2025 The go-wemix-wbft Authors
// This file is part of the go-wemix-wbft library.
//
// The go-wemix-wbft library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Regression tests for GovStaking security fixes. Each test intentionally
// triggers the attack precondition and asserts the contract enforces the
// correct outcome post-fix. Side-effect preservation tests assert the
// normal/legitimate paths still succeed.
//
// Scenarios covered:
//
//   * claim() — operator can only claim on behalf of the staker they are
//     registered for; cross-staker drains must fail.
//   * claim() — operator's own delegation reward on another staker remains
//     claimable after the scope fix.
//   * withdraw() — shortening the unbonding period via governance must not
//     let users drain earlier, still-locked credentials.
//   * withdraw() — when a single account accumulates staker credentials and
//     delegator credentials with different unbonding periods, the older
//     locked credential must not be drained via the bulk path.
//   * withdraw() — liveness: every matured credential is reachable even when
//     an earlier credential is still locked. Locked / empty slots are
//     skipped; the withdrawalIndex pointer advances to the oldest live slot.
//   * withdraw() — explicit-count mode succeeds when enough matures exist,
//     and reverts with "insufficient withdrawable credentials" when the
//     count cannot be met by currently-mature credentials.
//   * receive() — only a registered active rewardee vault may send coin to
//     GovStaking; direct EOA sends revert.
//
// Related documents:
//   - docs/secrity-review/fix-report-govstaking-security.md
//   - docs/secrity-review/govstaking-logic-flow.md
//   - docs/secrity-review/govstaking-withdraw-liveness-evaluation.md

package test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	govwbft "github.com/ethereum/go-ethereum/wemixgov/governance-wbft"
	"github.com/stretchr/testify/require"
)

// -----------------------------------------------------------------------------
// claim() — operator permission scope
// -----------------------------------------------------------------------------

// TestClaim_OperatorCannotDrainAnotherStakersReward asserts that an operator
// cannot claim rewards from a staker they are not registered as the operator
// for. The previous implementation used `isOperator(msg.sender)`, which
// returned true for any registered operator and allowed cross-staker drains.
// The corrected implementation uses `stakerByOperator[msg.sender] == _staker`
// so that operator-on-behalf-of-staker claims are scoped per-staker.
func TestClaim_OperatorCannotDrainAnotherStakersReward(t *testing.T) {
	var (
		ctx          = context.TODO()
		feeRate      = new(big.Int).SetUint64(1000)
		minStaking   = towei(500000)
		rewardAmount = towei(10)

		sX = NewTestStaker() // victim staker (operator OpX)
		sY = NewTestStaker() // attacker's own staker (operator OpY)
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
		sY.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g)
	defer g.backend.Close()

	stateDB := &TestStateDB{
		getState: func(addr common.Address, hash common.Hash) common.Hash {
			value, _ := g.backend.Client().StorageAt(ctx, addr, hash, nil)
			return common.BytesToHash(value)
		},
	}

	// Register both stakers; each pays its own minStaking from its operator's balance.
	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.RegisterStaker(t, sY, minStaking, feeRate))
	require.NoError(t, err)

	// Accrue reward on victim sX (send coin to sX's rewardee vault).
	distributeReward(t, g, stateDB, rewardAmount, sX.Staker.Address)

	// Precondition: OpY has no stake or delegation in sX.
	stakerOfOpY := govwbft.StakerByOperator(TestGovStakingAddress, stateDB, sY.Operator.Address)
	require.Equal(t, sY.Staker.Address, stakerOfOpY, "OpY must only be registered as operator of sY")

	t.Run("cross-staker claim must revert", func(t *testing.T) {
		// stakerByOperator[OpY] == sY.Staker != sX.Staker therefore
		// _user = msg.sender = OpY. OpY has neither stakingAmount nor
		// pendingReward under sX, so the "no reward to claim" guard fires.
		err := g.ExpectedFail(g.Claim(t, sY.Operator, sX.Staker.Address, false))
		ExpectedRevert(t, err, "no reward to claim")

		// Victim's pending reward must remain claimable by the legitimate
		// operator (proves no drain happened).
		beforeOpX := g.balanceAt(t, ctx, sX.Operator.Address, nil)
		receipt, err := g.ExpectedOk(g.Claim(t, sX.Operator, sX.Staker.Address, false))
		require.NoError(t, err)
		afterOpX := g.balanceAt(t, ctx, sX.Operator.Address, nil)
		gasCost := calcTxGasCost(receipt)
		// OpX gets the full reward minus gas.
		expectedDelta := new(big.Int).Sub(rewardAmount, gasCost)
		require.Equal(t, expectedDelta, new(big.Int).Sub(afterOpX, beforeOpX),
			"legitimate operator receives reward net of gas")
	})
}

// TestClaim_OperatorCanClaimOwnDelegationOnAnotherStaker asserts that an
// operator who also holds a legitimate delegation to a different staker can
// still claim their own delegation rewards. The scope fix must not break the
// cross-staker delegation case where the call is legitimate (operator is
// acting as a plain delegator on a staker they do not operate).
func TestClaim_OperatorCanClaimOwnDelegationOnAnotherStaker(t *testing.T) {
	var (
		ctx          = context.TODO()
		feeRate      = new(big.Int).SetUint64(1000)
		minStaking   = towei(500000)
		delegateAmt  = towei(100000)
		rewardAmount = towei(10)

		sX = NewTestStaker() // operator OpX; OpX will delegate to sY
		sY = NewTestStaker() // operator OpY
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
		sY.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g)
	defer g.backend.Close()

	stateDB := &TestStateDB{
		getState: func(addr common.Address, hash common.Hash) common.Hash {
			value, _ := g.backend.Client().StorageAt(ctx, addr, hash, nil)
			return common.BytesToHash(value)
		},
	}

	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.RegisterStaker(t, sY, minStaking, feeRate))
	require.NoError(t, err)

	// OpX (operator of sX) delegates to sY. Allowed because
	// delegate() only forbids self-delegation by sY's own operator.
	_, err = g.ExpectedOk(g.Delegate(t, sX.Operator, sY.Staker.Address, delegateAmt))
	require.NoError(t, err)

	// Accrue reward on sY; OpX's delegation now has pending reward.
	distributeReward(t, g, stateDB, rewardAmount, sY.Staker.Address)

	// OpX calls claim(sY, false). stakerByOperator[OpX]=sX.Staker, compared
	// against _staker=sY.Staker, so _user = msg.sender = OpX. OpX has
	// stakingAmount > 0 in userRewardInfo[sY][OpX] → claim succeeds and pays
	// OpX their delegator reward.
	beforeOpX := g.balanceAt(t, ctx, sX.Operator.Address, nil)
	receipt, err := g.ExpectedOk(g.Claim(t, sX.Operator, sY.Staker.Address, false))
	require.NoError(t, err)
	afterOpX := g.balanceAt(t, ctx, sX.Operator.Address, nil)

	// OpX's net balance must increase (some reward minus gas).
	gasCost := calcTxGasCost(receipt)
	require.True(t, afterOpX.Cmp(new(big.Int).Sub(beforeOpX, gasCost)) > 0,
		"OpX received at least some delegation reward net of gas")
}

// -----------------------------------------------------------------------------
// withdraw() — safety under governance-shortened unbonding period
// -----------------------------------------------------------------------------

// TestWithdraw_SafetyUnderGovernanceShortenedUnbonding asserts that shortening
// the staker unbonding period via governance after credentials are already in
// flight does NOT let the earlier (still-locked) credential be drained. The
// per-credential expiry check inside the loop prevents early drain regardless
// of mode; locked credentials are simply skipped.
func TestWithdraw_SafetyUnderGovernanceShortenedUnbonding(t *testing.T) {
	var (
		feeRate    = new(big.Int).SetUint64(1000)
		minStaking = towei(500000)

		sX = NewTestStaker()
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g) // unbondingPeriodStaker = 604800s (7d)
	defer g.backend.Close()

	// Register + top up so there is something to unstake while staying active.
	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)
	topUp := towei(10)
	_, err = g.ExpectedOk(g.Stake(t, sX.Operator, topUp))
	require.NoError(t, err)

	// credential[0] with the ORIGINAL long period (7d).
	_, err = g.ExpectedOk(g.Unstake(t, sX.Operator, towei(5)))
	require.NoError(t, err)

	// Governance shortens unbondingPeriodStaker to 1 hour.
	const shortPeriod int64 = 3600
	g.backend.CommitWithState(&params.GovContracts{
		GovConfig: &params.GovContract{
			Address: TestGovConfigAddress,
			Version: govwbft.GOV_CONTRACT_VERSION_1,
			Params: map[string]string{
				govwbft.GOV_CONFIG_PARAM_MINIMUM_STAKING:     minStaking.String(),
				govwbft.GOV_CONFIG_PARAM_MAXIMUM_STAKING:     MAX_UINT_128.String(),
				govwbft.GOV_CONFIG_PARAM_UNBONDING_STAKER:    big.NewInt(shortPeriod).String(),
				govwbft.GOV_CONFIG_PARAM_UNBONDING_DELEGATOR: "259200",
				govwbft.GOV_CONFIG_PARAM_FEE_PRECISION:       "10000",
				govwbft.GOV_CONFIG_PARAM_CHANGE_FEE_DELAY:    "604800",
			},
		},
	}, nil)

	// credential[1] with the SHORTER period, created after the governance change.
	_, err = g.ExpectedOk(g.Unstake(t, sX.Operator, towei(1)))
	require.NoError(t, err)

	// Advance past the short period — credential[1] is mature, credential[0] is not.
	g.adjustTime(time.Duration(shortPeriod+60) * time.Second)

	t.Run("explicit count=2 reverts because credential[0] is still locked", func(t *testing.T) {
		// Requested 2 matures but only 1 is actually mature (credential[1]).
		// The scan completes with processed=1 < count=2 → revert.
		// Critically, credential[0] is never drained.
		err := g.ExpectedFail(g.Withdraw(t, sX.Operator, common.Big2))
		ExpectedRevert(t, err, "insufficient withdrawable credentials")
	})

	t.Run("auto withdraw drains credential[1], leaves credential[0] locked", func(t *testing.T) {
		// Liveness: the user's 1-hour credential is accessible even though
		// credential[0] is still locked. VUL-005 is resolved by skipping
		// (not breaking on) locked credentials in auto mode.
		receipt, err := g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
		require.NoError(t, err)
		events := findEvents("Withdrawn", receipt.Logs)
		require.Len(t, events, 1, "exactly credential[1] must be drained")
		// The emitted storage index must be 1 (the short-period credential).
		require.Zero(t, events[0]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(1)),
			"Withdrawn event index must reflect the actual storage key")
	})

	t.Run("after credential[0] matures, auto withdraw drains it", func(t *testing.T) {
		// Wait the remaining 7d to mature credential[0].
		g.adjustTime(time.Duration(7*24*3600) * time.Second)

		receipt, err := g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
		require.NoError(t, err)
		events := findEvents("Withdrawn", receipt.Logs)
		require.Len(t, events, 1, "only credential[0] remained")
		require.Zero(t, events[0]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(0)))
	})
}

// -----------------------------------------------------------------------------
// withdraw() — staker/delegator mixed credentials (naturally occurring, no governance change)
// -----------------------------------------------------------------------------

// TestWithdraw_SafetyAndLivenessOnMixedStakerDelegatorCredentials reproduces
// the dual-role condition: the same EOA is the operator of one staker and a
// delegator to a different staker. Because unbondingPeriodStaker is longer
// than unbondingPeriodDelegator in standard deployments, credential[0] (from
// unstake) matures *later* than credential[1] (from undelegate).
//
// Safety: the earlier long-period credential cannot be drained early through
// any mode.
// Liveness: the later short-period credential is reachable as soon as it
// matures, even though credential[0] is still locked in front of it.
func TestWithdraw_SafetyAndLivenessOnMixedStakerDelegatorCredentials(t *testing.T) {
	var (
		feeRate     = new(big.Int).SetUint64(1000)
		minStaking  = towei(500000)
		delegateAmt = towei(50000)

		sX = NewTestStaker() // operator OpX plays dual role (unstake + undelegate)
		sY = NewTestStaker() // another staker; OpX delegates here
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
		sY.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g) // staker=604800 (7d), delegator=259200 (3d)
	defer g.backend.Close()

	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.RegisterStaker(t, sY, minStaking, feeRate))
	require.NoError(t, err)

	// Add some stake so we can unstake a portion without deactivating.
	_, err = g.ExpectedOk(g.Stake(t, sX.Operator, towei(20)))
	require.NoError(t, err)

	// credential[OpX][0]: from unstake → unbondingPeriodStaker (7d)
	_, err = g.ExpectedOk(g.Unstake(t, sX.Operator, towei(10)))
	require.NoError(t, err)

	// OpX delegates to sY, then undelegates → credential[OpX][1]: unbondingPeriodDelegator (3d)
	_, err = g.ExpectedOk(g.Delegate(t, sX.Operator, sY.Staker.Address, delegateAmt))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.Undelegate(t, sX.Operator, sY.Staker.Address, delegateAmt))
	require.NoError(t, err)

	// Advance 3d + slack → credential[1] mature, credential[0] still locked.
	g.adjustTime(time.Duration(259200+60) * time.Second)

	t.Run("explicit count=2 reverts because only 1 mature", func(t *testing.T) {
		// Scan finds 1 mature; requested 2 → "insufficient..." revert.
		// Safety: credential[0] is never drained.
		err := g.ExpectedFail(g.Withdraw(t, sX.Operator, common.Big2))
		ExpectedRevert(t, err, "insufficient withdrawable credentials")
	})

	t.Run("auto withdraw drains the mature delegator credential, skips locked staker credential", func(t *testing.T) {
		// Liveness: credential[1] is reachable even though credential[0] is
		// still locked. Only the mature one is emitted.
		receipt, err := g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
		require.NoError(t, err)
		events := findEvents("Withdrawn", receipt.Logs)
		require.Len(t, events, 1, "only credential[1] (mature) must be drained")
		require.Zero(t, events[0]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(1)),
			"drained credential must be the delegator one at index 1")
	})

	t.Run("after credential[0] matures, auto withdraw drains it", func(t *testing.T) {
		// Wait the remaining 4 days to mature credential[0] (7d total for staker period).
		g.adjustTime(time.Duration(4*24*3600) * time.Second)

		receipt, err := g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
		require.NoError(t, err)

		events := findEvents("Withdrawn", receipt.Logs)
		require.Len(t, events, 1, "only credential[0] remained from previous partial withdrawal")
		require.Zero(t, events[0]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(0)))
	})
}

// -----------------------------------------------------------------------------
// Side-effect preservation — withdraw()
// -----------------------------------------------------------------------------

// TestWithdraw_ExplicitAllMatureStillWorks verifies that the post-fix
// withdraw() remains fully backward-compatible when every credential in the
// requested range is mature.
func TestWithdraw_ExplicitAllMatureStillWorks(t *testing.T) {
	var (
		feeRate    = new(big.Int).SetUint64(1000)
		minStaking = towei(500000)

		sX = NewTestStaker()
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g)
	defer g.backend.Close()

	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.Stake(t, sX.Operator, towei(30)))
	require.NoError(t, err)

	// Three sequential unstakes — all with the same unbondingPeriodStaker, so
	// withdrawableTime is monotonically increasing.
	for i := 0; i < 3; i++ {
		_, err = g.ExpectedOk(g.Unstake(t, sX.Operator, towei(10)))
		require.NoError(t, err)
	}

	// Advance beyond the staker unbonding period.
	g.adjustTime(time.Duration(604800+60) * time.Second)

	receipt, err := g.ExpectedOk(g.Withdraw(t, sX.Operator, big.NewInt(3)))
	require.NoError(t, err)
	require.Len(t, findEvents("Withdrawn", receipt.Logs), 3)
}

// TestWithdraw_AutoModeAllImmatureYieldsNothing verifies that auto mode
// returns no events when no credential is mature yet.
func TestWithdraw_AutoModeAllImmatureYieldsNothing(t *testing.T) {
	var (
		feeRate    = new(big.Int).SetUint64(1000)
		minStaking = towei(500000)

		sX = NewTestStaker()
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g)
	defer g.backend.Close()

	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.Stake(t, sX.Operator, towei(20)))
	require.NoError(t, err)

	// Two unstakes — both immediately in locked state.
	_, err = g.ExpectedOk(g.Unstake(t, sX.Operator, towei(10)))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.Unstake(t, sX.Operator, towei(5)))
	require.NoError(t, err)

	g.adjustTime(time.Hour)

	receipt, err := g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
	require.NoError(t, err)
	require.Empty(t, findEvents("Withdrawn", receipt.Logs),
		"auto mode yields nothing when nothing is mature yet")

	// Now mature both and auto-drain.
	g.adjustTime(time.Duration(604800+60) * time.Second)
	receipt, err = g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
	require.NoError(t, err)
	require.Len(t, findEvents("Withdrawn", receipt.Logs), 2,
		"auto mode drains all mature credentials")
}

// TestReceive_OnlyActiveRewardeeAllowed covers the GovStaking `receive()`
// guard: only an active staker's rewardee vault may send coin to the staking
// contract (used by the internal restake flow). Direct sends by an EOA or
// other contract must revert, preventing balance inflation attacks.
func TestReceive_OnlyActiveRewardeeAllowed(t *testing.T) {
	var (
		feeRate    = new(big.Int).SetUint64(1000)
		minStaking = towei(500000)
		sX         = NewTestStaker()
		attacker   = NewEOA()
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
		attacker.Address:    {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g)
	defer g.backend.Close()

	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)

	// Direct coin send from an EOA must revert.
	govStakingAddr := TestGovStakingAddress
	tx, txErr := TransferCoin(
		g.backend.Client(),
		NewTxOptsWithValue(t, attacker, nil),
		big.NewInt(1),
		&govStakingAddr,
	)
	// TransferCoin may fail at gas estimation (pre-submit revert) or at commit
	// time — either way, expectedFail normalizes both into a revert error.
	_, err = expectedFail(g.backend, tx, txErr)
	ExpectedRevert(t, err, "only an active rewardee can send coin")
}

// TestWithdraw_InsufficientCountReverts verifies explicit mode rejects a
// request that cannot be fully satisfied by currently-mature credentials.
// The whole call must revert atomically; no partial processing.
func TestWithdraw_InsufficientCountReverts(t *testing.T) {
	var (
		feeRate    = new(big.Int).SetUint64(1000)
		minStaking = towei(500000)
		sX         = NewTestStaker()
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g)
	defer g.backend.Close()

	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.Stake(t, sX.Operator, towei(10)))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.Unstake(t, sX.Operator, towei(5)))
	require.NoError(t, err)

	// count=5 but only 1 credential exists → insufficient matures.
	err = g.ExpectedFail(g.Withdraw(t, sX.Operator, big.NewInt(5)))
	ExpectedRevert(t, err, "insufficient withdrawable credentials")
}

// -----------------------------------------------------------------------------
// withdraw() — liveness new coverage (Option C)
// -----------------------------------------------------------------------------

// TestWithdraw_PointerAdvancesToOldestLiveSlot verifies that after an auto
// withdrawal that drains some but not all credentials, the withdrawalIndex
// pointer ends up pointing to the oldest still-live (locked) slot — not to
// an already-emptied slot and not past a live slot.
func TestWithdraw_PointerAdvancesToOldestLiveSlot(t *testing.T) {
	var (
		feeRate     = new(big.Int).SetUint64(1000)
		minStaking  = towei(500000)
		delegateAmt = towei(50000)

		sX = NewTestStaker()
		sY = NewTestStaker()
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
		sY.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g) // staker=604800 (7d), delegator=259200 (3d)
	defer g.backend.Close()

	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.RegisterStaker(t, sY, minStaking, feeRate))
	require.NoError(t, err)

	_, err = g.ExpectedOk(g.Stake(t, sX.Operator, towei(20)))
	require.NoError(t, err)

	// credentials[OpX][0] — 7d staker period (will remain locked after first withdraw)
	_, err = g.ExpectedOk(g.Unstake(t, sX.Operator, towei(10)))
	require.NoError(t, err)

	// credentials[OpX][1] — 3d delegator period (will mature first)
	_, err = g.ExpectedOk(g.Delegate(t, sX.Operator, sY.Staker.Address, delegateAmt))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.Undelegate(t, sX.Operator, sY.Staker.Address, delegateAmt))
	require.NoError(t, err)

	// Advance 3d + slack → credentials[1] mature, credentials[0] locked.
	g.adjustTime(time.Duration(259200+60) * time.Second)

	// Drain credentials[1] via auto mode.
	receipt, err := g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
	require.NoError(t, err)
	require.Len(t, findEvents("Withdrawn", receipt.Logs), 1)

	// A second auto call at the same block must not emit anything (credentials[0]
	// is still locked, credentials[1] is empty now).
	receipt, err = g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
	require.NoError(t, err)
	require.Empty(t, findEvents("Withdrawn", receipt.Logs),
		"second auto call must yield 0 events: pointer should have advanced to locked credentials[0]")

	// After credentials[0] matures, the same auto call drains it.
	g.adjustTime(time.Duration(4*24*3600) * time.Second)
	receipt, err = g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
	require.NoError(t, err)
	events := findEvents("Withdrawn", receipt.Logs)
	require.Len(t, events, 1)
	require.Zero(t, events[0]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(0)))
}

// TestWithdraw_ExplicitPartialMatureDrainsOnlyRequested verifies that when
// more matures are available than requested, explicit count drains exactly
// `count` matures and leaves the rest live.
func TestWithdraw_ExplicitPartialMatureDrainsOnlyRequested(t *testing.T) {
	var (
		feeRate    = new(big.Int).SetUint64(1000)
		minStaking = towei(500000)

		sX = NewTestStaker()
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g)
	defer g.backend.Close()

	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.Stake(t, sX.Operator, towei(30)))
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		_, err = g.ExpectedOk(g.Unstake(t, sX.Operator, towei(10)))
		require.NoError(t, err)
	}

	// All 3 become mature.
	g.adjustTime(time.Duration(604800+60) * time.Second)

	// Explicit count=2 must drain only 2 credentials.
	receipt, err := g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big2))
	require.NoError(t, err)
	events := findEvents("Withdrawn", receipt.Logs)
	require.Len(t, events, 2)
	require.Zero(t, events[0]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(0)))
	require.Zero(t, events[1]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(1)))

	// Remaining credential[2] still withdrawable in a subsequent call.
	receipt, err = g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big1))
	require.NoError(t, err)
	events = findEvents("Withdrawn", receipt.Logs)
	require.Len(t, events, 1)
	require.Zero(t, events[0]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(2)))
}

// TestWithdraw_ExplicitAdvancesPointerPastEmptyHoles verifies that when an
// earlier withdraw call leaves empty (previously-drained) slots between
// withdrawalIndex and credentialIndex, a later explicit-mode withdraw that
// hits its requested count inside that prefix still scans past the empty
// holes to find the next still-live slot and parks withdrawalIndex there.
//
// Regression guard against two classes of mistake:
//
//  1. Breaking unconditionally once _processed == _withdrawalCount would
//     leave _oldestAlive unset, fast-forward withdrawalIndex to
//     credentialIndex, and strand later live credentials (funds lock-up).
//  2. Setting _oldestAlive to an empty slot would park withdrawalIndex on a
//     drained index, wasting gas on future scans.
//
// Scenario construction produces credentials = [0:mature, 1:empty, 2:locked]
// with withdrawalIndex == 0, then calls withdraw(count=1). Correct behavior:
// drain slot 0, skip slot 1, recognize slot 2 as the oldest still-live slot
// and advance withdrawalIndex to 2.
func TestWithdraw_ExplicitAdvancesPointerPastEmptyHoles(t *testing.T) {
	var (
		feeRate     = new(big.Int).SetUint64(1000)
		minStaking  = towei(500000)
		delegateAmt = towei(50000)

		sX = NewTestStaker()
		sY = NewTestStaker()
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
		sY.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g) // staker=604800 (7d), delegator=259200 (3d)
	defer g.backend.Close()

	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.RegisterStaker(t, sY, minStaking, feeRate))
	require.NoError(t, err)

	_, err = g.ExpectedOk(g.Stake(t, sX.Operator, towei(30)))
	require.NoError(t, err)

	// credentials[OpX][0] — 7d staker period.
	_, err = g.ExpectedOk(g.Unstake(t, sX.Operator, towei(10)))
	require.NoError(t, err)

	// credentials[OpX][1] — 3d delegator period (matures first).
	_, err = g.ExpectedOk(g.Delegate(t, sX.Operator, sY.Staker.Address, delegateAmt))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.Undelegate(t, sX.Operator, sY.Staker.Address, delegateAmt))
	require.NoError(t, err)

	// Advance 3d + slack → credentials[0] still locked, credentials[1] mature.
	g.adjustTime(time.Duration(259200+60) * time.Second)

	// Drain credentials[1] via auto mode. State afterwards:
	//   credentials = [0:locked, 1:empty], withdrawalIndex = 0.
	receipt, err := g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
	require.NoError(t, err)
	events := findEvents("Withdrawn", receipt.Logs)
	require.Len(t, events, 1)
	require.Zero(t, events[0]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(1)))

	// Advance enough so credentials[0] also matures (total ~7d+ since unstake).
	g.adjustTime(time.Duration(604800-259200+60) * time.Second)

	// credentials[OpX][2] — brand-new 3d delegator credential (locked).
	_, err = g.ExpectedOk(g.Delegate(t, sX.Operator, sY.Staker.Address, delegateAmt))
	require.NoError(t, err)
	_, err = g.ExpectedOk(g.Undelegate(t, sX.Operator, sY.Staker.Address, delegateAmt))
	require.NoError(t, err)

	// State now: credentials = [0:mature, 1:empty, 2:locked],
	//            withdrawalIndex = 0, credentialIndex = 3.
	// Explicit withdraw(1) must drain credentials[0] only, skip the empty
	// credentials[1], and advance the pointer to credentials[2].
	receipt, err = g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big1))
	require.NoError(t, err)
	events = findEvents("Withdrawn", receipt.Logs)
	require.Len(t, events, 1, "explicit count=1 must drain exactly one credential")
	require.Zero(t, events[0]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(0)),
		"drained credential must be credentials[0]")

	// If the pointer had been fast-forwarded to credentialIndex (3), the
	// next call would revert with "no credential to withdraw". If it had
	// been parked on the empty credentials[1], the call would succeed but
	// also scan the empty slot. Either way, the post-fix pointer must sit
	// on credentials[2] (still locked) — auto mode at the same block must
	// not revert and must emit zero events.
	receipt, err = g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
	require.NoError(t, err)
	require.Empty(t, findEvents("Withdrawn", receipt.Logs),
		"auto mode on a fully-locked residual set must emit no events")

	// After credentials[2] matures the same auto call drains it, proving
	// the pointer is positioned to reach it.
	g.adjustTime(time.Duration(259200+60) * time.Second)
	receipt, err = g.ExpectedOk(g.Withdraw(t, sX.Operator, common.Big0))
	require.NoError(t, err)
	events = findEvents("Withdrawn", receipt.Logs)
	require.Len(t, events, 1)
	require.Zero(t, events[0]["withdrawalIndex"].(*big.Int).Cmp(big.NewInt(2)),
		"final drain must target credentials[2]")
}

// TestWithdraw_NoCredentialRevertsUnchanged verifies the empty-set guard is
// preserved (both pointers equal means no live credential).
func TestWithdraw_NoCredentialRevertsUnchanged(t *testing.T) {
	var (
		feeRate    = new(big.Int).SetUint64(1000)
		minStaking = towei(500000)

		sX = NewTestStaker()
	)

	g, err := NewGovWBFT(t, nil, types.GenesisAlloc{
		sX.Operator.Address: {Balance: new(big.Int).Mul(MAX_UINT_128, common.Big2)},
	})
	require.NoError(t, err)
	setWbftGovConfig(g)
	defer g.backend.Close()

	_, err = g.ExpectedOk(g.RegisterStaker(t, sX, minStaking, feeRate))
	require.NoError(t, err)

	// No unstake/undelegate yet; credentialIndex == withdrawalIndex == 0.
	err = g.ExpectedFail(g.Withdraw(t, sX.Operator, common.Big0))
	ExpectedRevert(t, err, "no credential to withdraw")

	err = g.ExpectedFail(g.Withdraw(t, sX.Operator, common.Big1))
	ExpectedRevert(t, err, "no credential to withdraw")
}
