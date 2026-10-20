package testhelper

import (
	"context"
	"fmt"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// TestPublicConsensusRetainsLegacyHeightSpread uses the public checkpoints.
// Matching historical hashes must not conceal a node over two blocks behind.
func TestPublicConsensusRetainsLegacyHeightSpread(t *testing.T) {
	raws, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/node/CT-NODE-009-head-hash-agreement.json"}, "../../presets/chain/stablenet-bp4-en1.json")
	if err != nil {
		t.Fatal(err)
	}
	original, err := dsl.ParseV2(raws[0])
	if err != nil {
		t.Fatal(err)
	}
	var windows [][]dsl.Statement
	var current []dsl.Statement
	for _, st := range original.Sequence {
		if st.Do == "waitBlock" || st.Expect == "blockAdvance" {
			continue
		}
		current = append(current, st)
		if st.Expect == "sameBlockHash" {
			windows = append(windows, current)
			current = nil
		}
	}
	if len(windows) != 2 || len(current) != 0 {
		t.Fatal("public case must retain both complete hash checkpoints")
	}
	fixtures := []struct {
		name  string
		heads [5]uint64
		want  session.TestStatus
	}{
		{"equal", [5]uint64{20, 20, 20, 20, 20}, session.StatusPass},
		{"two-block-boundary", [5]uint64{20, 21, 22, 20, 21}, session.StatusPass},
		{"non-anchor-extremes", [5]uint64{21, 20, 23, 21, 21}, session.StatusFail},
		{"stuck-at-genesis", [5]uint64{23, 23, 23, 23, 0}, session.StatusFail},
	}
	for lagger := 0; lagger < 5; lagger++ {
		heads := [5]uint64{23, 23, 23, 23, 23}
		heads[lagger] = 20
		fixtures = append(fixtures, struct {
			name  string
			heads [5]uint64
			want  session.TestStatus
		}{fmt.Sprintf("lagging-node%d", lagger+1), heads, session.StatusFail})
	}
	for checkpoint, statements := range windows {
		for _, f := range fixtures {
			t.Run(fmt.Sprintf("checkpoint%d/%s", checkpoint+1, f.name), func(t *testing.T) {
				urls := []string{}
				for _, height := range f.heads {
					n := &chainNode{head: height, hashes: map[string]string{"0x0": "0xsame-genesis", "0x14": "0xsame-block"}}
					urls = append(urls, n.serve(t).URL)
				}
				env := envOf(t, urls...)
				nodes := env.Nodes()
				nodes[4].Role = node.RoleEN
				env.PopulateNodeTable(node.NodeSet{Nodes: nodes})
				spec := original
				spec.Sequence = statements
				rec := &recordStub{}
				got, err := interp.NewInterpreter(deps()).Run(context.Background(), spec, env, rec)
				if err != nil || got != f.want {
					t.Fatalf("actual public checkpoint status=%s error=%v want=%s heads=%v assertions=%+v", got, err, f.want, f.heads, rec.asserts)
				}
			})
		}
	}
}
