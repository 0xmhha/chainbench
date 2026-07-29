// This file ports two multi-call system-contract read cases (from
// regression f-system-contracts f4-01 minter-status and f3-04
// validator-metadata), exercising uint/bool decoding and dynamic address-array
// decoding of a GovValidator return.
//
// # Test: minter-status-readable
//
// Intent:   the native-coin adapter answers isMinter(address) with a boolean word
//
//	(0 or 1) and minterAllowance(address) as a decodable uint256 — the reads
//	behind the GovMasterMinter configureMinter flow.
//
// Applies:  stablenet. Requires "rpc".
// Method:   eth_call isMinter(addr) and minterAllowance(addr); assert the first is
//
//	0 or 1 and the second decodes as a non-negative uint256.
//
// Pass:     isMinter is 0/1 and minterAllowance decodes.
//
// # Test: validator-metadata-readable
//
// Intent:   GovValidator.validatorList() returns the active validator addresses,
//
//	and each validatorToOperator(v) maps to a non-zero operator address.
//
// Applies:  stablenet. Requires "rpc".
// Method:   eth_call validatorList(), decode the address[]; for each validator
//
//	eth_call validatorToOperator(v) and assert the operator word is non-zero.
//
// Pass:     validatorList is non-empty and every validator maps to a non-zero
//
//	operator.
package anzeon

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "minter-status-readable",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           minterStatusReadable,
	})
	testkit.Register(testkit.Case{
		Name:         "validator-metadata-readable",
		Category:     "system-contracts",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           validatorMetadataReadable,
	})
}

func minterStatusReadable(t *testkit.T) {
	call := caller(t)
	isMinter, err := accounts.ReadUint(t.Ctx(), call, nativeCoinAdapter,
		"isMinter(address)", accounts.AddressArg(sampleAccount))
	t.NoErr(err, "isMinter")
	t.Truef(isMinter.Sign() == 0 || isMinter.Cmp(big.NewInt(1)) == 0,
		"isMinter returns 0 or 1 (got %s)", isMinter)

	allowance, err := accounts.ReadUint(t.Ctx(), call, nativeCoinAdapter,
		"minterAllowance(address)", accounts.AddressArg(sampleAccount))
	t.NoErr(err, "minterAllowance")
	t.Truef(allowance.Sign() >= 0, "minterAllowance decodes as a uint256 (got %s)", allowance)
}

func validatorMetadataReadable(t *testkit.T) {
	call := caller(t)
	ret, err := call(t.Ctx(), govValidator, accounts.EncodeCall("validatorList()"))
	t.NoErr(err, "validatorList")

	addrs, ok := decodeAddressArray(ret)
	t.Truef(ok && len(addrs) > 0, "validatorList() returns a non-empty address[] (got %q)", ret)

	for _, v := range addrs {
		op, err := accounts.ReadUint(t.Ctx(), call, govValidator,
			"validatorToOperator(address)", accounts.AddressArg(v))
		t.NoErr(err, "validatorToOperator")
		t.Truef(op.Sign() != 0, "validator %s maps to a non-zero operator", v)
	}
}

// decodeAddressArray decodes an ABI-encoded dynamic address[] return (offset,
// length, then one 32-byte word per address) into 0x-hex addresses.
func decodeAddressArray(hexRet string) ([]string, bool) {
	b, err := hex.DecodeString(strings.TrimPrefix(hexRet, "0x"))
	if err != nil || len(b) < 64 {
		return nil, false
	}
	off := new(big.Int).SetBytes(b[0:32]).Int64()
	if off < 0 || off+32 > int64(len(b)) {
		return nil, false
	}
	length := new(big.Int).SetBytes(b[off : off+32]).Int64()
	if length < 0 {
		return nil, false
	}
	addrs := make([]string, 0, length)
	for i := int64(0); i < length; i++ {
		start := off + 32 + i*32
		if start+32 > int64(len(b)) {
			return nil, false
		}
		word := b[start : start+32]
		addrs = append(addrs, "0x"+hex.EncodeToString(word[12:32]))
	}
	return addrs, true
}
