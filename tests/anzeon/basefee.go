// This file ports the anzeon base-fee gas policy floor and ceiling (from
// tests/regression c-anzeon c-06/c-07), read cases over the latest block's base
// fee.
//
// # Test: basefee-minimum
//
// Intent:   go-stablenet's anzeon gas policy floors the EIP-1559 base fee, so the
//
//	latest block's baseFeePerGas never drops below the minimum.
//
// Applies:  stablenet (the anzeon base-fee floor). Requires "rpc".
// Method:   read eth_getBlockByNumber("latest").baseFeePerGas and assert it is at
//
//	least the anzeon minimum (regression MIN_BASE_FEE_WEI).
//
// Pass:     baseFeePerGas >= the anzeon minimum.
//
// # Test: basefee-maximum
//
// Intent:   the anzeon policy also caps the base fee, so it never exceeds the
//
//	maximum (regression c-07 MAX_BASE_FEE, 20,000 Gwei).
//
// Applies:  stablenet. Requires "rpc".
// Method:   read the latest baseFeePerGas and assert it is at most the maximum.
// Pass:     baseFeePerGas <= the anzeon maximum.
package anzeon

import (
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

// minBaseFeeWei is the anzeon minimum base fee (regression lib/common.sh
// MIN_BASE_FEE_WEI).
var minBaseFeeWei, _ = new(big.Int).SetString("20000000000000", 10)

// maxBaseFeeWei is the anzeon maximum base fee (regression c-07 MAX_BASE_FEE,
// 20,000 Gwei).
var maxBaseFeeWei, _ = new(big.Int).SetString("20000000000000000", 10)

func init() {
	testkit.Register(testkit.Case{
		Name:         "basefee-minimum",
		Category:     "gas-policy",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           basefeeMinimum,
	})
	testkit.Register(testkit.Case{
		Name:         "basefee-maximum",
		Category:     "gas-policy",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           basefeeMaximum,
	})
}

func basefeeMinimum(t *testkit.T) {
	bf := latestBaseFee(t)
	t.Truef(bf.Cmp(minBaseFeeWei) >= 0, "baseFeePerGas %s < anzeon minimum %s", bf, minBaseFeeWei)
}

func basefeeMaximum(t *testkit.T) {
	bf := latestBaseFee(t)
	t.Truef(bf.Cmp(maxBaseFeeWei) <= 0, "baseFeePerGas %s > anzeon maximum %s", bf, maxBaseFeeWei)
}

// latestBaseFee reads and parses the latest block's baseFeePerGas.
func latestBaseFee(t *testkit.T) *big.Int {
	var block struct {
		BaseFeePerGas string `json:"baseFeePerGas"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &block, "latest", false), "eth_getBlockByNumber")
	bf, ok := new(big.Int).SetString(strings.TrimPrefix(block.BaseFeePerGas, "0x"), 16)
	t.Truef(ok, "baseFeePerGas %q is not hex", block.BaseFeePerGas)
	return bf
}
