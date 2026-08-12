package consensus_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/consensus" // register the case

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

const testMixHash = "0x1234000000000000000000000000000000000000000000000000000000000000"

// randaoMock answers the head-block reads WBFT-010 makes, with randomness fields
// controllable so both the pass and the missing-field paths can be exercised.
func randaoMock(t *testing.T, randao, mixHash string) string {
	t.Helper()
	srv := mockServer(t, func(method string, _ []any) any {
		switch method {
		case "eth_blockNumber":
			return "0x10"
		case "istanbul_getWbftExtraInfo":
			return map[string]any{"randaoReveal": randao}
		case "eth_getBlockByNumber":
			return map[string]any{"mixHash": mixHash}
		default:
			return nil
		}
	})
	return srv.URL
}

func TestRandaoMixDigestCase_Pass(t *testing.T) {
	if !registered("randao-and-mixdigest-present") {
		t.Fatal("randao-and-mixdigest-present not registered")
	}
	url := randaoMock(t, "0xabcd", testMixHash)
	ns, _ := node.AttachedSet("wbft", "local", []node.RPCEndpoint{{RPCURL: url}})
	if r := runCase(t, ns, "randao-and-mixdigest-present"); r.Status != testkit.StatusPass {
		t.Fatalf("status %s (%s)", r.Status, r.Message)
	}
}

// A zero mixHash means no randao mix was derived -> the case must fail.
func TestRandaoMixDigestCase_FailsOnZeroMix(t *testing.T) {
	url := randaoMock(t, "0xabcd", zeroMixHash)
	ns, _ := node.AttachedSet("wbft", "local", []node.RPCEndpoint{{RPCURL: url}})
	if r := runCase(t, ns, "randao-and-mixdigest-present"); r.Status != testkit.StatusFail {
		t.Fatalf("expected fail on zero mixHash, got %s", r.Status)
	}
}

// An empty randaoReveal means the WBFT randomness field is absent -> fail.
func TestRandaoMixDigestCase_FailsOnEmptyRandao(t *testing.T) {
	url := randaoMock(t, "0x", testMixHash)
	ns, _ := node.AttachedSet("wbft", "local", []node.RPCEndpoint{{RPCURL: url}})
	if r := runCase(t, ns, "randao-and-mixdigest-present"); r.Status != testkit.StatusFail {
		t.Fatalf("expected fail on empty randaoReveal, got %s", r.Status)
	}
}

func TestRandaoMixDigestCase_SkipsForeignChain(t *testing.T) {
	ns, _ := node.AttachedSet("ethereum", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"randao-and-mixdigest-present"}})
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("status %s, want skip on ethereum", r.Status)
		}
	}
}

const zeroMixHash = "0x0000000000000000000000000000000000000000000000000000000000000000"
