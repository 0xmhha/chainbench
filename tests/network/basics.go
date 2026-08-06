// Package network holds multi-node network test cases (ported from the legacy
// bash regression suite regression/a-ethereum/a1-*).
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
// # Test: block-progression
//
// Intent:   a launched network must be producing blocks — the head advances and
//
//	timestamps do not go backwards (ported from a-ethereum block-period).
//
// Applies:  all chains. Requires: the "rpc" capability.
// Method:   read the head and the two most recent blocks; assert the number
//
//	strictly increases and the timestamp is non-decreasing.
//
// Pass:     block N has a greater number and a >= timestamp than block N-1.
//
// These are chainbench TEST CODE (requirement #16): registered at init and run
// by the testrun phase against a live NodeSet, not by `go test` (the sibling
// _test.go validates registration/convention).
package network

import (
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/testkit"
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
	testkit.Register(testkit.Case{
		Name:         "block-progression",
		Category:     "network",
		RequiresCaps: []string{"rpc"},
		Fn:           blockProgression,
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

func blockProgression(t *testkit.T) {
	cli := t.Primary()
	var headHex string
	t.NoErr(cli.Call(t.Ctx(), "eth_blockNumber", &headHex), "eth_blockNumber")
	head, err := strconv.ParseUint(strings.TrimPrefix(headHex, "0x"), 16, 64)
	t.NoErr(err, "parse head")
	t.Truef(head >= 1, "chain has produced at least one block (head=%d)", head)

	latestNum, latestTS := blockNumTS(t, cli, head)
	prevNum, prevTS := blockNumTS(t, cli, head-1)
	t.Truef(latestNum > prevNum, "block number advances (%d > %d)", latestNum, prevNum)
	t.Truef(latestTS >= prevTS, "block timestamp is non-decreasing (%d >= %d)", latestTS, prevTS)
}

// blockNumTS reads a block's number and timestamp (as uint64) via eth_getBlockByNumber.
func blockNumTS(t *testkit.T, cli testkit.Client, n uint64) (uint64, uint64) {
	var blk struct {
		Number    string `json:"number"`
		Timestamp string `json:"timestamp"`
	}
	t.NoErr(cli.Call(t.Ctx(), "eth_getBlockByNumber", &blk, "0x"+strconv.FormatUint(n, 16), false), "eth_getBlockByNumber")
	num, _ := strconv.ParseUint(strings.TrimPrefix(blk.Number, "0x"), 16, 64)
	ts, _ := strconv.ParseUint(strings.TrimPrefix(blk.Timestamp, "0x"), 16, 64)
	return num, ts
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
