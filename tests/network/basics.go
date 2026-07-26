// Package network holds multi-node network test cases (ported from the legacy
// bash regression suite tests/regression/a-ethereum/a1-*).
//
// # Test: genesis-hash-agreement
//
// Intent:   every node must share the same genesis block 0 hash — the most basic
//
//	proof the nodes are on the same chain (a mismatched genesis is the
//	classic cause of a silent no-peer / fork).
//
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   read eth_getBlockByNumber(0x0) on every node; assert all hashes equal.
// Pass:     at least one node answers and all answering nodes share one hash.
//
// # Test: peers-connected
//
// Intent:   in a multi-node network every node must have at least one peer;
//
//	an isolated node cannot sync or reach consensus.
//
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   read net_peerCount on every node; assert > 0 (vacuous for a
//
//	single-node network, which has no peers to expect).
//
// Pass:     every node reports at least one peer.
//
// These are chainbench TEST CODE (requirement #16): registered at init and run
// by the testrun phase against a live NodeSet, not by `go test` (the sibling
// _test.go validates registration/convention).
package network

import (
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/pkg/testkit"
)

func init() {
	testkit.Register(testkit.Case{
		Name:         "genesis-hash-agreement",
		Category:     "network",
		RequiresCaps: []string{"rpc"},
		Fn:           genesisHashAgreement,
	})
	testkit.Register(testkit.Case{
		Name:         "peers-connected",
		Category:     "network",
		RequiresCaps: []string{"rpc"},
		Fn:           peersConnected,
	})
}

func genesisHashAgreement(t *testkit.T) {
	var first string
	answered := 0
	for _, n := range t.NodeSet().Nodes {
		var blk struct {
			Hash string `json:"hash"`
		}
		if err := t.Node(n.Index).Call(t.Ctx(), "eth_getBlockByNumber", &blk, "0x0", false); err != nil {
			t.Errorf("node%d eth_getBlockByNumber(0): %v", n.Index, err)
			continue
		}
		if blk.Hash == "" {
			continue
		}
		answered++
		if first == "" {
			first = blk.Hash
		}
		t.Equalf(blk.Hash, first, "node%d genesis hash matches", n.Index)
	}
	t.Truef(answered > 0, "at least one node returned genesis block 0")
}

func peersConnected(t *testkit.T) {
	nodes := t.NodeSet().Nodes
	if len(nodes) < 2 {
		return // single-node network has no peers to expect
	}
	for _, n := range nodes {
		var hexCount string
		if err := t.Node(n.Index).Call(t.Ctx(), "net_peerCount", &hexCount); err != nil {
			t.Errorf("node%d net_peerCount: %v", n.Index, err)
			continue
		}
		count, err := strconv.ParseUint(strings.TrimPrefix(hexCount, "0x"), 16, 64)
		if err != nil {
			t.Errorf("node%d peer count %q not hex: %v", n.Index, hexCount, err)
			continue
		}
		t.Truef(count > 0, "node%d has at least one peer (got %d)", n.Index, count)
	}
}
