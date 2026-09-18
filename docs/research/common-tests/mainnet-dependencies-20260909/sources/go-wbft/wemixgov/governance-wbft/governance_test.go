// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright 2025 The go-wemix-wbft Authors
// This file is part of the go-wemix-wbft library.

package govwbft

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// newTestGovContracts builds a minimal, valid GovContracts fixture using the
// production default addresses.
func newTestGovContracts(withNCP bool, ncps string) *params.GovContracts {
	gc := &params.GovContracts{
		GovConfig: &params.GovContract{
			Address: params.DefaultGovConfigAddress,
			Version: params.DefaultGovVersion,
			Params: map[string]string{
				GOV_CONFIG_PARAM_MINIMUM_STAKING:     "500000000000000000000000",
				GOV_CONFIG_PARAM_MAXIMUM_STAKING:     "100000000000000000000000000",
				GOV_CONFIG_PARAM_UNBONDING_STAKER:    "604800",
				GOV_CONFIG_PARAM_UNBONDING_DELEGATOR: "259200",
				GOV_CONFIG_PARAM_FEE_PRECISION:       "10000",
				GOV_CONFIG_PARAM_CHANGE_FEE_DELAY:    "604800",
				// govCouncil optional — left unset
			},
		},
		GovStaking: &params.GovContract{
			Address: params.DefaultGovStakingAddress,
			Version: params.DefaultGovVersion,
		},
		GovRewardeeImp: &params.GovContract{
			Address: params.DefaultGovRewardeeImpAddress,
			Version: params.DefaultGovVersion,
		},
	}
	if withNCP {
		gc.GovNCP = &params.GovContract{
			Address: params.DefaultGovNCPAddress,
			Version: params.DefaultGovVersion,
			Params: map[string]string{
				GOV_NCP_PARAM_NCPS: ncps,
			},
		}
	}
	return gc
}

// codeFor finds the bytecode assignment for addr in a CodeParam slice.
func codeFor(t *testing.T, codes []params.CodeParam, addr common.Address) string {
	t.Helper()
	for _, c := range codes {
		if c.Address == addr {
			return c.Code
		}
	}
	t.Fatalf("no CodeParam found for %s", addr.Hex())
	return ""
}

// stateFor looks up the value assigned to (addr, key) in a StateParam slice.
func stateFor(t *testing.T, states []params.StateParam, addr common.Address, key common.Hash) (common.Hash, bool) {
	t.Helper()
	for _, s := range states {
		if s.Address == addr && s.Key == key {
			return s.Value, true
		}
	}
	return common.Hash{}, false
}

// TestGetGovContractsTransition_Full verifies the full initialization image
// emitted for a production-like GovContracts fixture.
func TestGetGovContractsTransition_Full(t *testing.T) {
	ncpAddrs := []common.Address{
		common.HexToAddress("0xaa"),
		common.HexToAddress("0xbb"),
	}
	ncpStr := ncpAddrs[0].Hex() + " , " + ncpAddrs[1].Hex() // extra whitespace tests splitAndTrim
	gc := newTestGovContracts(true, ncpStr)

	st, err := GetGovContractsTransition(gc)
	require.NoError(t, err)
	require.NotNil(t, st)

	// --- Codes --------------------------------------------------------------
	require.NotEmpty(t, codeFor(t, st.Codes, params.DefaultGovConfigAddress),
		"GovConfig bytecode must be deployed")
	require.NotEmpty(t, codeFor(t, st.Codes, params.DefaultGovStakingAddress),
		"GovStaking bytecode must be deployed")
	require.NotEmpty(t, codeFor(t, st.Codes, params.DefaultGovRewardeeImpAddress),
		"GovRewardeeImp bytecode must be deployed")
	require.NotEmpty(t, codeFor(t, st.Codes, params.DefaultGovNCPAddress),
		"GovNCP bytecode must be deployed")

	// --- GovConfig states ---------------------------------------------------
	for slotKey, wantStr := range map[string]string{
		SLOT_GOV_CONFIG_MINIMUM_STAKING:     "500000000000000000000000",
		SLOT_GOV_CONFIG_MAXIMUM_STAKING:     "100000000000000000000000000",
		SLOT_GOV_CONFIG_UNBONDING_STAKER:    "604800",
		SLOT_GOV_CONFIG_UNBONDING_DELEGATOR: "259200",
		SLOT_GOV_CONFIG_FEE_PRECISION:       "10000",
		SLOT_GOV_CONFIG_CHANGE_FEE_DELAY:    "604800",
	} {
		got, ok := stateFor(t, st.States, params.DefaultGovConfigAddress, common.HexToHash(slotKey))
		require.True(t, ok, "slot %s missing", slotKey)
		want, _ := new(big.Int).SetString(wantStr, 10)
		require.Equal(t, common.BigToHash(want), got, "slot %s", slotKey)
	}

	// govCouncil is unset → no state entry for its slot.
	_, ok := stateFor(t, st.States, params.DefaultGovConfigAddress, common.HexToHash(SLOT_GOV_CONFIG_GOV_COUNCIL))
	require.False(t, ok, "govCouncil slot must be absent when not configured")

	// --- GovStaking initial states -----------------------------------------
	blsPoP, ok := stateFor(t, st.States, params.DefaultGovStakingAddress, common.HexToHash(SLOT_BLS_POP_PRECOMPILED_ADDRESS))
	require.True(t, ok)
	require.Equal(t, common.BytesToHash(params.BLSPoPPrecompileAddress.Bytes()), blsPoP)

	cfgAddr, ok := stateFor(t, st.States, params.DefaultGovStakingAddress, common.HexToHash(SLOT_GOV_CONFIG_ADDRESS))
	require.True(t, ok)
	require.Equal(t, common.BytesToHash(params.DefaultGovConfigAddress.Bytes()), cfgAddr)

	rewardeeAddr, ok := stateFor(t, st.States, params.DefaultGovStakingAddress, common.HexToHash(SLOT_GOV_REWARDEE_IMP_ADDRESS))
	require.True(t, ok)
	require.Equal(t, common.BytesToHash(params.DefaultGovRewardeeImpAddress.Bytes()), rewardeeAddr)

	// --- NCP list replay ---------------------------------------------------
	reader := newMockStateReader()
	for _, s := range st.States {
		if s.Address == params.DefaultGovNCPAddress {
			reader.set(s.Address, s.Key, s.Value)
		}
	}
	require.Equal(t, ncpAddrs, NCPList(params.DefaultGovNCPAddress, reader))
	require.Equal(t, uint64(2), NCPLength(params.DefaultGovNCPAddress, reader))
}

// TestGetGovContractsTransition_GovCouncilSet ensures that a non-zero
// govCouncil param writes to the council slot.
func TestGetGovContractsTransition_GovCouncilSet(t *testing.T) {
	gc := newTestGovContracts(false, "")
	council := common.HexToAddress("0x1234")
	gc.GovConfig.Params[GOV_CONFIG_PARAM_GOV_COUNCIL] = council.Hex()

	st, err := GetGovContractsTransition(gc)
	require.NoError(t, err)

	got, ok := stateFor(t, st.States, params.DefaultGovConfigAddress, common.HexToHash(SLOT_GOV_CONFIG_GOV_COUNCIL))
	require.True(t, ok)
	require.Equal(t, common.BytesToHash(council.Bytes()), got)
}

// TestGetGovContractsTransition_InvalidParams covers the numeric parse
// failures returned by GetGovContractsTransition.
func TestGetGovContractsTransition_InvalidParams(t *testing.T) {
	for _, invalidKey := range []string{
		GOV_CONFIG_PARAM_MINIMUM_STAKING,
		GOV_CONFIG_PARAM_MAXIMUM_STAKING,
		GOV_CONFIG_PARAM_UNBONDING_STAKER,
		GOV_CONFIG_PARAM_UNBONDING_DELEGATOR,
		GOV_CONFIG_PARAM_FEE_PRECISION,
		GOV_CONFIG_PARAM_CHANGE_FEE_DELAY,
	} {
		t.Run(invalidKey, func(t *testing.T) {
			gc := newTestGovContracts(false, "")
			gc.GovConfig.Params[invalidKey] = "not-a-number"
			_, err := GetGovContractsTransition(gc)
			require.ErrorContains(t, err, "invalid gov config params")
		})
	}
}

// TestGetGovContractsTransition_NCPEmpty covers the guard that blocks an empty
// NCP set when GovNCP is configured.
func TestGetGovContractsTransition_NCPEmpty(t *testing.T) {
	gc := newTestGovContracts(true, "   ,  , ")
	_, err := GetGovContractsTransition(gc)
	require.ErrorContains(t, err, "no initial NCPs provided")
}

// TestCheckGovContractVersions covers the version whitelist that CheckGovContractVersions
// consults for initialization.
func TestCheckGovContractVersions(t *testing.T) {
	t.Run("accepts all known versions", func(t *testing.T) {
		gc := newTestGovContracts(true, common.HexToAddress("0xaa").Hex())
		require.NoError(t, checkGovContractVersions(gc))
	})

	t.Run("rejects unknown GovConfig version", func(t *testing.T) {
		gc := newTestGovContracts(false, "")
		gc.GovConfig.Version = "v-unknown"
		require.Error(t, checkGovContractVersions(gc))
	})

	t.Run("rejects unknown GovStaking version", func(t *testing.T) {
		gc := newTestGovContracts(false, "")
		gc.GovStaking.Version = "v-unknown"
		require.Error(t, checkGovContractVersions(gc))
	})

	t.Run("rejects unknown GovRewardeeImp version", func(t *testing.T) {
		gc := newTestGovContracts(false, "")
		gc.GovRewardeeImp.Version = "v-unknown"
		require.Error(t, checkGovContractVersions(gc))
	})

	t.Run("rejects unknown GovNCP version", func(t *testing.T) {
		gc := newTestGovContracts(true, common.HexToAddress("0xaa").Hex())
		gc.GovNCP.Version = "v-unknown"
		require.Error(t, checkGovContractVersions(gc))
	})
}

// TestSplitAndTrim covers the CSV parser used for the NCP list param.
func TestSplitAndTrim(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{" a , b ,  c ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
		{"  ", nil},
		{"", nil},
		{"onlyone", []string{"onlyone"}},
	}
	for _, c := range cases {
		require.Equal(t, c.want, splitAndTrim(c.in, ","), "input=%q", c.in)
	}
}
