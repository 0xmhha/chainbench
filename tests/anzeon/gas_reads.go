// This file ports the anzeon gas-API read cases (from regression g-api
// g2-01/g2-02/g2-04), which exercise the go-stablenet backend's Anzeon gas-fee
// branch: eth_gasPrice and eth_maxPriorityFeePerGas are derived from the header
// WBFTExtra.gasTip reported by istanbul_getWbftExtraInfo.
//
// # Test: gas-price-equals-basefee-plus-tip
//
// Intent:   on the Anzeon backend branch eth_gasPrice equals the latest block's
//
//	baseFeePerGas plus the header WBFTExtra.gasTip.
//
// Applies:  stablenet. Requires "rpc".
// Method:   read eth_gasPrice, the latest block's number+baseFeePerGas, and
//
//	istanbul_getWbftExtraInfo(number).gasTip; assert gasPrice == baseFee + tip.
//
// Pass:     eth_gasPrice == baseFeePerGas + gasTip.
//
// # Test: max-priority-fee-equals-gastip
//
// Intent:   eth_maxPriorityFeePerGas equals the header WBFTExtra.gasTip.
// Applies:  stablenet. Requires "rpc".
// Method:   read eth_maxPriorityFeePerGas and the latest block's
//
//	istanbul_getWbftExtraInfo.gasTip; assert they are equal.
//
// Pass:     eth_maxPriorityFeePerGas == gasTip.
//
// # Test: estimate-gas-token-transfer
//
// Intent:   eth_estimateGas for a native-coin-adapter transfer(address,uint256)
//
//	call costs more than a bare 21000-gas value transfer (it runs contract
//	code), so the estimate exceeds 21000.
//
// Applies:  stablenet. Requires "rpc".
// Method:   eth_estimateGas for transfer(recipient, 1000) calldata to the
//
//	native-coin adapter; assert the estimate is > 21000.
//
// Pass:     estimate > 21000.
package anzeon

import (
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// estimateTarget is an arbitrary transfer recipient used only to build the
// transfer(address,uint256) calldata for the gas estimate.
const estimateTarget = "0x00000000000000000000000000000000C0FFEE09"

func init() {
	testkit.Register(testkit.Case{
		Name:         "gas-price-equals-basefee-plus-tip",
		Category:     "gas-policy",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           gasPriceEqualsBasefeePlusTip,
	})
	testkit.Register(testkit.Case{
		Name:         "max-priority-fee-equals-gastip",
		Category:     "gas-policy",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           maxPriorityFeeEqualsGasTip,
	})
	testkit.Register(testkit.Case{
		Name:         "estimate-gas-token-transfer",
		Category:     "gas-policy",
		ChainCompat:  []string{"stablenet"},
		RequiresCaps: []string{"rpc"},
		Fn:           estimateGasTokenTransfer,
	})
}

func gasPriceEqualsBasefeePlusTip(t *testkit.T) {
	var gasPriceHex string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_gasPrice", &gasPriceHex), "eth_gasPrice")

	var block struct {
		Number        string `json:"number"`
		BaseFeePerGas string `json:"baseFeePerGas"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &block, "latest", false), "eth_getBlockByNumber")

	baseFee := parseBig(block.BaseFeePerGas)
	tip := headerGasTip(t, block.Number)
	expected := new(big.Int).Add(baseFee, tip)
	t.Equalf(parseBig(gasPriceHex).String(), expected.String(),
		"eth_gasPrice == baseFee(%s) + gasTip(%s)", baseFee, tip)
}

func maxPriorityFeeEqualsGasTip(t *testkit.T) {
	var mpfHex string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_maxPriorityFeePerGas", &mpfHex), "eth_maxPriorityFeePerGas")

	var block struct {
		Number string `json:"number"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &block, "latest", false), "eth_getBlockByNumber")

	tip := headerGasTip(t, block.Number)
	t.Equalf(parseBig(mpfHex).String(), tip.String(),
		"eth_maxPriorityFeePerGas == header WBFTExtra.gasTip")
}

func estimateGasTokenTransfer(t *testkit.T) {
	data := accounts.EncodeCall("transfer(address,uint256)",
		accounts.AddressArg(estimateTarget), big.NewInt(1000).Bytes())
	var estHex string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_estimateGas", &estHex, map[string]any{
		"from": sampleAccount,
		"to":   nativeCoinAdapter,
		"data": data,
	}), "eth_estimateGas")
	est := parseBig(estHex)
	t.Truef(est.Cmp(big.NewInt(21000)) > 0,
		"eth_estimateGas for a native-coin transfer (%s) exceeds 21000", est)
}

// headerGasTip reads istanbul_getWbftExtraInfo(blockHex).gasTip as a big.Int.
// The field may be hex- or decimal-encoded, so parseBig handles both.
func headerGasTip(t *testkit.T, blockHex string) *big.Int {
	var extra struct {
		GasTip string `json:"gasTip"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getWbftExtraInfo", &extra, blockHex),
		"istanbul_getWbftExtraInfo")
	return parseBig(extra.GasTip)
}

// parseBig parses a JSON-RPC numeric string that may be 0x-hex or decimal (the
// WBFTExtra.gasTip field is not consistently hex). Unparseable input yields 0.
func parseBig(s string) *big.Int {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0)
	}
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		if v, ok := new(big.Int).SetString(s[2:], 16); ok {
			return v
		}
		return big.NewInt(0)
	}
	if v, ok := new(big.Int).SetString(s, 10); ok {
		return v
	}
	return big.NewInt(0)
}
