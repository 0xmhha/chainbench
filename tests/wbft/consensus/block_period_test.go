package consensus_test

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/consensus" // register the case

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func TestBlockPeriodCase_Passes(t *testing.T) {
	const head = 0x64 // 100
	const base = 1_700_000_000
	srv := mockServer(t, func(method string, params []any) any {
		switch method {
		case "eth_blockNumber":
			return fmt.Sprintf("0x%x", head)
		case "eth_getBlockByNumber":
			// Timestamp advances exactly 1s per block -> 1s period.
			n := uint64(0)
			if len(params) > 0 {
				if s, ok := params[0].(string); ok {
					n, _ = strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
				}
			}
			return map[string]any{"timestamp": fmt.Sprintf("0x%x", base+n)}
		default:
			return nil
		}
	})
	ns, _ := node.AttachedSet("wbft", "local", []node.RPCEndpoint{{RPCURL: srv.URL}})

	if !registered("block-period-one-second") {
		t.Fatal("block-period-one-second not registered")
	}
	if r := runCase(t, ns, "block-period-one-second"); r.Status != testkit.StatusPass {
		t.Fatalf("block-period-one-second: %+v", r)
	}
}

func TestBlockPeriodCase_SkipForeignChain(t *testing.T) {
	ns, _ := node.AttachedSet("ethereum", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"block-period-one-second"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on ethereum, got %+v", rep.Results)
	}
}
