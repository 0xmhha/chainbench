package testhelper

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// TestGasPriceSum: the suggested price must be the head's base fee plus the
// suggested tip; a price that is merely positive, or off by the tip, fails.
func TestGasPriceSum(t *testing.T) {
	block := map[string]any{"number": "0x9", "hash": "0xb", "baseFeePerGas": "0x3b9aca00", "timestamp": "0x1", "transactions": []any{}}
	for _, c := range []struct {
		price string
		pass  bool
	}{
		{"0x77359400", true},  // 1 gwei base + 1 gwei tip
		{"0x3b9aca00", false}, // base alone
		{"0x1", false},        // positive, and nothing more
	} {
		srv := mockRPC(t, map[string]any{
			"eth_blockNumber": "0x9", "eth_getBlockByNumber": block,
			"eth_maxPriorityFeePerGas": "0x3b9aca00", "eth_gasPrice": c.price,
		})
		d := deps()
		as, ok := d.Actions.Assertion(assertGasPriceSum)
		if !ok {
			t.Fatal("gasPriceIsBaseFeePlusTip is not registered")
		}
		r, err := as.Check(context.Background(), &interp.AssertCtx{Deps: &d, On: []node.Node{{Index: 1, RPCURL: srv.URL}}, Spec: map[string]any{"assert": assertGasPriceSum}})
		if err != nil {
			t.Fatalf("price %s: %v", c.price, err)
		}
		if r.Pass != c.pass {
			t.Errorf("price %s: pass = %v, want %v (%s)", c.price, r.Pass, c.pass, r.Source)
		}
	}
}
