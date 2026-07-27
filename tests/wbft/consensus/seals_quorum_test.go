package consensus_test

import (
	"context"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/consensus" // register the cases

	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

var quorumCases = []string{"commit-signers-quorum", "wbft-seals-quorum", "prev-seals-quorum"}

// fourValidators is the sealer set the seal mocks report (quorum = 3).
var fourValidators = []string{
	"0xc17d493883eaa3b4cceb0f214b273392d562f9d8",
	"0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c",
	"0x1a2b3c4d5e6f70819293a4b5c6d7e8f90a1b2c3d",
	"0x9f8e7d6c5b4a39281706f5e4d3c2b1a09f8e7d6c",
}

func sealMockSet(t *testing.T) node.NodeSet {
	seal := func(n int) map[string]any {
		return map[string]any{"sealers": fourValidators[:n], "signature": "0xabcdef"}
	}
	srv := mockServer(t, func(method string, _ []any) any {
		switch method {
		case "istanbul_getValidators":
			return fourValidators
		case "istanbul_getCommitSignersFromBlock":
			return map[string]any{
				"Author":     "0xc17d493883eaa3b4cceb0f214b273392d562f9d8",
				"Committers": fourValidators[:3],
			}
		case "istanbul_getWbftExtraInfo":
			return map[string]any{
				"committedSeal":     seal(4),
				"preparedSeal":      seal(4),
				"prevCommittedSeal": seal(4),
				"prevPreparedSeal":  seal(3),
			}
		default:
			return nil
		}
	})
	return consensusSet(node.Node{Role: node.RoleValidator, RPCURL: srv.URL})
}

func TestSealQuorumCases(t *testing.T) {
	ns := sealMockSet(t)
	for _, name := range quorumCases {
		if !registered(name) {
			t.Errorf("case %q not registered", name)
			continue
		}
		if r := runCase(t, ns, name); r.Status != testkit.StatusPass {
			t.Errorf("%s: %+v", name, r)
		}
	}
}

func TestSealQuorumCases_SkipMissingConsensusCap(t *testing.T) {
	// rpc-only set (no "consensus") must gate the consensus cases out.
	ns := node.NodeSet{Chain: "wbft", Network: "local", Capabilities: []string{"rpc"},
		Nodes: []node.Node{{Index: 1, Role: node.RoleValidator, RPCURL: "http://x"}}}
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: quorumCases})
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip without consensus cap", r.Name, r.Status)
		}
	}
}

func TestSealQuorumCases_SkipForeignChain(t *testing.T) {
	ns := node.NodeSet{Chain: "ethereum", Network: "local", Capabilities: []string{"rpc", "consensus"},
		Nodes: []node.Node{{Index: 1, Role: node.RoleValidator, RPCURL: "http://x"}}}
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: quorumCases})
	for _, r := range rep.Results {
		if r.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on ethereum", r.Name, r.Status)
		}
	}
}
