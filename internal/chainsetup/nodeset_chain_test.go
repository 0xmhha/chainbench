package chainsetup

import (
	"encoding/json"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestNodeSetRetainsEachRecordedBinaryChain(t *testing.T) {
	w := &Workspace{state: State{Chain: "wemix", BinaryChains: map[string]string{"next": "wbft"}, Nodes: []node.Record{{Index: 1}, {Index: 2, Binary: "next"}, {Index: 3, Binary: "other"}}}}
	encoded, err := json.Marshal(w.NodeSet())
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Nodes []struct {
			Index int
			Chain string
		}
	}
	if err := json.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	want := []string{"wemix", "wbft", "wemix"}
	for i, n := range result.Nodes {
		if n.Chain != want[i] {
			t.Errorf("node%d chain=%q, want %q", n.Index, n.Chain, want[i])
		}
	}
}
