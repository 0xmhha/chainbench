package mcp

import (
	"math"
	"testing"
)

func TestComposeChainIDMatchesExactCLIInputs(t *testing.T) {
	for _, c := range []struct {
		value any
		want  int64
	}{{0, 0}, {77773, 77773}, {"00077773", 77773}, {"0X12FCD", 77773}, {"9223372036854775807", math.MaxInt64}} {
		in, err := composeInput(map[string]any{"chainId": c.value}, nil, "")
		if err != nil || in.ChainID != c.want {
			t.Fatal("chain ID changed", c, err, in.ChainID)
		}
	}
	for _, bad := range []any{nil, true, 1.5, -1, float64(9007199254740992), "9223372036854775808", "0x8000000000000000", ""} {
		if _, err := composeInput(map[string]any{"chainId": bad}, nil, ""); err == nil {
			t.Fatal("lossy chain ID accepted", bad)
		}
	}
}
