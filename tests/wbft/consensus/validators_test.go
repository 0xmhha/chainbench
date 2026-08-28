package consensus_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/consensus" // register the cases

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func TestValidatorSetCount_Passes(t *testing.T) {
	// Engine reports 4 validators; the set runs 3 validator-role nodes -> pass.
	srv := mockServer(t, func(method string, _ []any) any {
		if method == "istanbul_getValidators" {
			return fourValidators
		}
		return nil
	})
	ns := consensusSet(
		node.Node{Role: node.RoleValidator, RPCURL: srv.URL},
		node.Node{Role: node.RoleValidator, RPCURL: srv.URL},
		node.Node{Role: node.RoleValidator, RPCURL: srv.URL},
		node.Node{Role: node.RoleEndpoint, RPCURL: srv.URL},
	)
	if r := runCase(t, ns, "validator-set-count"); r.Status != testkit.StatusPass {
		t.Fatalf("validator-set-count: %+v", r)
	}
}

func TestValidatorCases_SkipForeignChain(t *testing.T) {
	names := []string{"validator-set-count"}
	ns := node.NodeSet{Chain: "wemix", Network: "local", Capabilities: []string{"rpc", "consensus"},
		Nodes: []node.Node{{Index: 1, Role: node.RoleValidator, RPCURL: "http://x"}}}
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: names})
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on wemix", r.Name, r.Status)
		}
	}
}
