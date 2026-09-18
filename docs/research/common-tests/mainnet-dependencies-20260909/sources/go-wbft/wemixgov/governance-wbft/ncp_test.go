// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright 2025 The go-wemix-wbft Authors
// This file is part of the go-wemix-wbft library.

package govwbft

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// applyStateParams replays a []StateParam sequence onto a mockStateReader.
// Used to verify initializeNCP() produces a storage image that matches the
// runtime read helpers (NCPList / IsNCP / NCPAt / NCPLength).
func applyStateParams(reader *mockStateReader, params []stateParam) {
	for _, p := range params {
		reader.set(p.addr, p.key, p.value)
	}
}

// stateParam is a local mirror of params.StateParam without the package cycle.
type stateParam struct {
	addr  common.Address
	key   common.Hash
	value common.Hash
}

// flattenInitializeNCP converts []params.StateParam into []stateParam so
// applyStateParams can consume it without cross-package imports.
func flattenInitializeNCP(govNCP common.Address, ncps []common.Address) []stateParam {
	raw := initializeNCP(govNCP, ncps)
	out := make([]stateParam, 0, len(raw))
	for _, r := range raw {
		out = append(out, stateParam{addr: r.Address, key: r.Key, value: r.Value})
	}
	return out
}

// TestInitializeNCP_Unique verifies that a unique NCP list is written into
// storage correctly and can be read back via the public helpers.
func TestInitializeNCP_Unique(t *testing.T) {
	govNCP := common.HexToAddress("0x1003")
	ncps := []common.Address{
		common.HexToAddress("0x0a"),
		common.HexToAddress("0x0b"),
		common.HexToAddress("0x0c"),
	}

	reader := newMockStateReader()
	applyStateParams(reader, flattenInitializeNCP(govNCP, ncps))

	require.Equal(t, uint64(3), NCPLength(govNCP, reader))
	require.Equal(t, ncps, NCPList(govNCP, reader))
	for _, n := range ncps {
		require.True(t, IsNCP(govNCP, reader, n))
	}
	require.False(t, IsNCP(govNCP, reader, common.HexToAddress("0xfe")))

	// lastId = 3 at slot SLOT_NCP_LAST_ID
	require.Equal(t,
		common.BigToHash(big.NewInt(3)),
		reader.GetState(govNCP, common.HexToHash(SLOT_NCP_LAST_ID)),
	)
}

// TestInitializeNCP_Duplicated verifies that repeated addresses are skipped,
// i.e. the resulting set is deduped.
func TestInitializeNCP_Duplicated(t *testing.T) {
	govNCP := common.HexToAddress("0x1003")
	dup := common.HexToAddress("0x0a")
	other := common.HexToAddress("0x0b")

	reader := newMockStateReader()
	applyStateParams(reader, flattenInitializeNCP(govNCP, []common.Address{dup, dup, other, dup}))

	require.Equal(t, uint64(2), NCPLength(govNCP, reader))
	require.Equal(t, []common.Address{dup, other}, NCPList(govNCP, reader))
}

// TestInitializeNCP_Empty verifies that an empty NCP slice produces no state
// changes (allowing callers to short-circuit).
func TestInitializeNCP_Empty(t *testing.T) {
	govNCP := common.HexToAddress("0x1003")
	require.Empty(t, initializeNCP(govNCP, nil))
	require.Empty(t, initializeNCP(govNCP, []common.Address{}))
}

// TestNCPAt covers the index-based accessor.
func TestNCPAt(t *testing.T) {
	govNCP := common.HexToAddress("0x1003")
	ncps := []common.Address{
		common.HexToAddress("0x0a"),
		common.HexToAddress("0x0b"),
	}

	reader := newMockStateReader()
	applyStateParams(reader, flattenInitializeNCP(govNCP, ncps))

	for i, want := range ncps {
		require.Equal(t, want, NCPAt(govNCP, reader, big.NewInt(int64(i))))
	}
}
