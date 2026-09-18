// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright 2025 The go-wemix-wbft Authors
// This file is part of the go-wemix-wbft library.

package govwbft

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"
)

// mockStateReader is a simple in-memory StateReader keyed by (addr, slot).
type mockStateReader struct {
	store map[common.Address]map[common.Hash]common.Hash
}

func newMockStateReader() *mockStateReader {
	return &mockStateReader{
		store: make(map[common.Address]map[common.Hash]common.Hash),
	}
}

func (m *mockStateReader) GetState(addr common.Address, key common.Hash) common.Hash {
	if slots, ok := m.store[addr]; ok {
		return slots[key]
	}
	return common.Hash{}
}

func (m *mockStateReader) set(addr common.Address, key common.Hash, value common.Hash) {
	if _, ok := m.store[addr]; !ok {
		m.store[addr] = make(map[common.Hash]common.Hash)
	}
	m.store[addr][key] = value
}

// TestCalculateMappingSlot verifies the solidity mapping slot formula:
//
//	keccak256(abi.encode(key, baseSlot))
func TestCalculateMappingSlot(t *testing.T) {
	baseSlot := common.HexToHash("0x0a")
	key := common.HexToAddress("0x00000000000000000000000000000000deadbeef")

	expected := sha3.NewLegacyKeccak256()
	expected.Write(common.LeftPadBytes(key.Bytes(), 32))
	expected.Write(baseSlot.Bytes())
	want := common.BytesToHash(expected.Sum(nil))

	require.Equal(t, want, CalculateMappingSlot(baseSlot, key),
		"CalculateMappingSlot must equal keccak256(pad(key) || slot)")
}

// TestCalculateDynamicSlot verifies the solidity dynamic array element formula:
//
//	keccak256(baseSlot) + index
func TestCalculateDynamicSlot(t *testing.T) {
	baseSlot := common.HexToHash("0x04")

	h := sha3.NewLegacyKeccak256()
	h.Write(common.LeftPadBytes(baseSlot.Bytes(), 32))
	start := new(big.Int).SetBytes(h.Sum(nil))

	for i := range 5 {
		got := CalculateDynamicSlot(baseSlot, big.NewInt(int64(i)))
		want := common.BigToHash(new(big.Int).Add(start, big.NewInt(int64(i))))
		require.Equal(t, want, got, "index=%d", i)
	}
}

// TestIncrementHash covers the small adder that produces sibling slots.
func TestIncrementHash(t *testing.T) {
	base := common.HexToHash("0x42")
	require.Equal(t, common.HexToHash("0x42"), IncrementHash(base, big.NewInt(0)))
	require.Equal(t, common.HexToHash("0x43"), IncrementHash(base, big.NewInt(1)))
	require.Equal(t, common.HexToHash("0x46"), IncrementHash(base, big.NewInt(4)))
}

// TestEnumerableSet_Length_Contains_Values_At exercises the storage-backed
// AddressSet against a synthetic layout identical to
// OpenZeppelin EnumerableSet.AddressSet.
func TestEnumerableSet_Length_Contains_Values_At(t *testing.T) {
	addr := common.HexToAddress("0xabcd")
	base := common.HexToHash("0x04")
	index := IncrementHash(base, big.NewInt(1))
	reader := newMockStateReader()

	// Put two items into the set.
	items := []common.Address{
		common.HexToAddress("0x01"),
		common.HexToAddress("0x02"),
	}
	for i, v := range items {
		// values[i] = v
		reader.set(addr, CalculateDynamicSlot(base, big.NewInt(int64(i))), common.BytesToHash(v.Bytes()))
		// indexes[v] = i+1
		reader.set(addr, CalculateMappingSlot(index, v), common.BigToHash(big.NewInt(int64(i+1))))
	}
	// length = 2
	reader.set(addr, base, common.BigToHash(big.NewInt(int64(len(items)))))

	set := NewAddressSet(base)
	require.Equal(t, uint64(2), set.Length(reader, addr))
	require.True(t, set.Contains(reader, addr, items[0]))
	require.True(t, set.Contains(reader, addr, items[1]))
	require.False(t, set.Contains(reader, addr, common.HexToAddress("0x99")))
	require.Equal(t, items, set.Values(reader, addr))
	require.Equal(t, items[1], set.At(reader, addr, big.NewInt(1)))
}

// TestGetBytes_Short verifies the <=31 byte short form of Solidity dynamic bytes.
func TestGetBytes_Short(t *testing.T) {
	addr := common.HexToAddress("0xaaaa")
	slot := common.HexToHash("0x07")
	reader := newMockStateReader()

	payload := []byte("hello-short-bytes") // 17 bytes
	var raw [32]byte
	copy(raw[:len(payload)], payload)
	raw[31] = byte(len(payload) << 1) // length<<1, low bit clear = short form
	reader.set(addr, slot, common.BytesToHash(raw[:]))

	got := GetBytes(reader, addr, slot)
	require.Equal(t, payload, got)
}

// TestGetBytes_Long verifies the >31 byte long form: slot stores (length<<1)|1,
// and the actual data lives at keccak(slot) + offset.
func TestGetBytes_Long(t *testing.T) {
	addr := common.HexToAddress("0xbbbb")
	slot := common.HexToHash("0x08")
	reader := newMockStateReader()

	payload := make([]byte, 50)
	for i := range payload {
		payload[i] = byte('A' + (i % 26))
	}

	// length encoding
	lengthEncoded := new(big.Int).Or(
		new(big.Int).Lsh(big.NewInt(int64(len(payload))), 1),
		big.NewInt(1),
	)
	reader.set(addr, slot, common.BigToHash(lengthEncoded))

	// split payload into 32-byte chunks at keccak(slot)+i
	var chunk0, chunk1 [32]byte
	copy(chunk0[:], payload[:32])
	copy(chunk1[:], payload[32:])
	reader.set(addr, CalculateDynamicSlot(slot, big.NewInt(0)), chunk0)
	reader.set(addr, CalculateDynamicSlot(slot, big.NewInt(1)), chunk1)

	got := GetBytes(reader, addr, slot)
	require.Equal(t, payload, got)
}

// TestGetBytes_Empty covers the empty-slot fast path.
func TestGetBytes_Empty(t *testing.T) {
	addr := common.HexToAddress("0xcccc")
	slot := common.HexToHash("0x09")
	reader := newMockStateReader() // no data
	require.Equal(t, []byte{}, GetBytes(reader, addr, slot))
}

// TestHashToAddress verifies the trivial conversion helper.
func TestHashToAddress(t *testing.T) {
	addr := common.HexToAddress("0x00000000000000000000000000000000cafebabe")
	require.Equal(t, addr, HashToAddress(common.BytesToHash(addr.Bytes())))
}
