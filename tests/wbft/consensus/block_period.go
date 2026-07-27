// # Test: block-period-one-second
//
// Intent:   wbft is configured for a 1-second block period; consecutive block
//
//	timestamps must advance by exactly that period, proving steady-state
//	block production without stalls or bursts (ported from
//	tests/regression/b-wbft/b-01-block-period.sh).
//
// Applies:  stablenet, wbft. Requires: "rpc".
// Method:   read a window of consecutive recent blocks via
//
//	eth_getBlockByNumber and diff their timestamps.
//
// Pass:     each adjacent pair of block timestamps differs by exactly 1 second.
//
// This is chainbench TEST CODE (requirement #16): registered at init and run by
// the testrun phase against a live NodeSet (the sibling _test.go validates
// registration and runs it against a mock node).
package consensus

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "block-period-one-second",
		Category:     "consensus",
		ChainCompat:  []string{"stablenet", "wbft"},
		RequiresCaps: []string{"rpc"},
		Fn:           blockPeriodOneSecond,
	})
}

// blockPeriod is the configured wbft block period in seconds.
const blockPeriod = 1

func blockPeriodOneSecond(t *testkit.T) {
	cli := t.Primary()
	head, err := cli.BlockNumber(t.Ctx())
	t.NoErr(err, "eth_blockNumber")
	t.Truef(head >= 2, "need at least 2 blocks to measure a period (got %d)", head)

	// Sample a window of consecutive blocks ending at the head.
	const window = 5
	start := uint64(1)
	if head > window {
		start = head - window
	}

	var prev uint64
	havePrev := false
	for n := start; n <= head; n++ {
		var blk struct {
			Timestamp string `json:"timestamp"`
		}
		t.NoErr(cli.Call(t.Ctx(), "eth_getBlockByNumber", &blk, fmt.Sprintf("0x%x", n), false),
			"eth_getBlockByNumber")
		ts, ok := parseHexU64(blk.Timestamp)
		t.Truef(ok, "block %d timestamp %q parses as hex", n, blk.Timestamp)
		if havePrev {
			t.Equalf(ts-prev, uint64(blockPeriod), "block %d->%d period", n-1, n)
		}
		prev = ts
		havePrev = true
	}
}

func parseHexU64(s string) (uint64, bool) {
	v, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
	return v, err == nil
}
