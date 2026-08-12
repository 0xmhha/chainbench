package consensus_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/consensus" // register the cases

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// istanbulReadsMock answers the rpc-only istanbul read methods with values that
// make node-address-returned, wbft-extra-info-fields and istanbul-status-fields
// pass.
func istanbulReadsMock(t *testing.T) *node.NodeSet {
	t.Helper()
	srv := mockServer(t, func(method string, _ []any) any {
		switch method {
		case "istanbul_nodeAddress":
			return "0xc17d493883eaa3b4cceb0f214b273392d562f9d8"
		case "istanbul_getWbftExtraInfo":
			return map[string]any{
				"gasTip":        "0x0",
				"committedSeal": map[string]any{"sealers": []string{}, "signature": "0x00"},
				"preparedSeal":  map[string]any{"sealers": []string{}, "signature": "0x00"},
			}
		case "istanbul_status":
			return map[string]any{
				"sealerActivity": map[string]any{},
				"author":         "0xc17d493883eaa3b4cceb0f214b273392d562f9d8",
				"blockRange":     []any{"0x1", "0x64"},
				"roundStats":     map[string]any{},
			}
		default:
			return nil
		}
	})
	ns, _ := node.AttachedSet("wbft", "local", []node.RPCEndpoint{{RPCURL: srv.URL}})
	return &ns
}

func TestIstanbulReadCases(t *testing.T) {
	ns := *istanbulReadsMock(t)
	for _, name := range []string{"node-address-returned", "wbft-extra-info-fields", "istanbul-status-fields"} {
		if !registered(name) {
			t.Errorf("case %q not registered", name)
			continue
		}
		if r := runCase(t, ns, name); r.Status != testkit.StatusPass {
			t.Errorf("%s: %+v", name, r)
		}
	}
}

func TestIstanbulReadCases_SkipForeignChain(t *testing.T) {
	ns, _ := node.AttachedSet("ethereum", "local", []node.RPCEndpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{
		Names: []string{"node-address-returned", "wbft-extra-info-fields", "istanbul-status-fields"},
	})
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on ethereum", r.Name, r.Status)
		}
	}
}
